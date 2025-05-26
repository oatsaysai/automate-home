package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TuyaConfig holds the configuration for Tuya API
type TuyaConfig struct {
	AccessID  string
	AccessKey string
	BaseURL   string
}

// TokenResponse represents the access token response from Tuya API
type TokenResponse struct {
	Result struct {
		AccessToken  string `json:"access_token"`
		ExpireTime   int64  `json:"expire_time"`
		RefreshToken string `json:"refresh_token"`
		UID          string `json:"uid"`
	} `json:"result"`
	Success bool   `json:"success"`
	T       int64  `json:"t"`
	Tid     string `json:"tid"`
}

// DeviceCommand represents a single command to send to a device
type DeviceCommand struct {
	Code  string      `json:"code"`
	Value interface{} `json:"value"`
}

// DeviceCommandRequest represents the request payload for device commands
type DeviceCommandRequest struct {
	Commands []DeviceCommand `json:"commands"`
}

// DeviceCommandResponse represents the response from device command API
type DeviceCommandResponse struct {
	Result  bool   `json:"result"`
	Success bool   `json:"success"`
	T       int64  `json:"t"`
	Tid     string `json:"tid"`
}

// TuyaClient represents the Tuya API client
type TuyaClient struct {
	config      TuyaConfig
	accessToken string
	httpClient  *http.Client
}

// NewTuyaClient creates a new Tuya API client
func NewTuyaClient(accessID, accessKey, baseURL string) *TuyaClient {
	// Create a custom transport with TLS configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, // Set to true only for development/testing
			MinVersion:         tls.VersionTLS12,
			// Use system certificate pool
			RootCAs: nil, // nil means use system's root CA set
		},
	}

	return &TuyaClient{
		config: TuyaConfig{
			AccessID:  accessID,
			AccessKey: accessKey,
			BaseURL:   baseURL,
		},
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// generateNonce generates a random nonce string
func (c *TuyaClient) generateNonce() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// sha256Hash generates SHA256 hash of the input
func (c *TuyaClient) sha256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// hmacSHA256 generates HMAC-SHA256 signature
func (c *TuyaClient) hmacSHA256(key, data string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}

// buildStringToSign creates the string to sign for the request
func (c *TuyaClient) buildStringToSign(method, uri, headers, body string) string {
	contentSha256 := c.sha256Hash([]byte(body))
	return fmt.Sprintf("%s\n%s\n%s\n%s", method, contentSha256, headers, uri)
}

// generateSignature creates the signature for the request
func (c *TuyaClient) generateSignature(method, uri, headers, body, timestamp, nonce string) string {
	stringToSign := c.buildStringToSign(method, uri, headers, body)
	signStr := c.config.AccessID + c.accessToken + timestamp + nonce + stringToSign
	return c.hmacSHA256(c.config.AccessKey, signStr)
}

// makeRequest makes an HTTP request to Tuya API
func (c *TuyaClient) makeRequest(method, endpoint, body string) (*http.Response, error) {
	requestURL := c.config.BaseURL + endpoint

	// Parse URL to extract path and query
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}

	// Build URI with sorted query parameters
	uri := parsedURL.Path
	if parsedURL.RawQuery != "" {
		form, err := url.ParseQuery(parsedURL.RawQuery)
		if err == nil {
			keys := make([]string, 0, len(form))
			for key := range form {
				keys = append(keys, key)
			}
			if len(keys) > 0 {
				uri += "?"
				sort.Strings(keys)
				for i, keyName := range keys {
					value := form.Get(keyName)
					uri += keyName + "=" + value
					if i < len(keys)-1 {
						uri += "&"
					}
				}
			}
		}
	}

	// Generate timestamp and nonce
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	nonce := c.generateNonce()

	// Create request
	req, err := http.NewRequest(method, requestURL, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("client_id", c.config.AccessID)
	req.Header.Set("sign", c.generateSignature(method, uri, "", body, timestamp, nonce))
	req.Header.Set("t", timestamp)
	req.Header.Set("sign_method", "HMAC-SHA256")
	req.Header.Set("nonce", nonce)

	if c.accessToken != "" {
		req.Header.Set("access_token", c.accessToken)
	}

	return c.httpClient.Do(req)
}

// GetAccessToken retrieves an access token from Tuya API
func (c *TuyaClient) GetAccessToken() (*TokenResponse, error) {
	resp, err := c.makeRequest("GET", "/v1.0/token?grant_type=1", "")
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !tokenResp.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	// Store the access token for future requests
	c.accessToken = tokenResp.Result.AccessToken

	return &tokenResp, nil
}

// SendDeviceCommand sends commands to a specific device
func (c *TuyaClient) SendDeviceCommand(deviceID string, commands []DeviceCommand) (*DeviceCommandResponse, error) {
	// Construct the endpoint URL
	endpoint := fmt.Sprintf("/v1.0/devices/%s/commands", deviceID)

	// Create the request payload
	request := DeviceCommandRequest{
		Commands: commands,
	}

	// Marshal the request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Make the POST request
	resp, err := c.makeRequest("POST", endpoint, string(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Unmarshal the response
	var commandResp DeviceCommandResponse
	if err := json.Unmarshal(body, &commandResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !commandResp.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &commandResp, nil
}

// SetInsecureSkipVerify configures whether to skip TLS certificate verification
// WARNING: This should only be used for development/testing environments
func (c *TuyaClient) SetInsecureSkipVerify(skip bool) {
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = skip
	}
}
