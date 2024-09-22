package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

var ch = make(chan int64)

func main() {

	go func() {
		for {
			err := openWebPage(ch)
			if err != nil {
				log.Printf("err: %v\n", err)
			}
			time.Sleep(1000 * time.Millisecond)
		}
	}()

	http.HandleFunc("/play-scene", playSceneHandler)

	log.Println("Server started at http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalln("Error starting server:", err)
	}
}

type PlaySceneParams struct {
	Scene int64 `json:"scene"`
}

func playSceneHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var params PlaySceneParams
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ch <- params.Scene

	fmt.Fprintf(w, "Play scene: %+v\n", params.Scene)
}

func openWebPage(ch chan int64) error {

	log.Println("Begin open web page")

	// create ctx
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	url := os.Getenv("HOST")
	username := os.Getenv("USER")
	password := os.Getenv("PASS")
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))

	var result string
	err := chromedp.Run(
		ctx,
		setHeadersAndNavigate(
			url,
			map[string]interface{}{
				"Authorization": authHeader,
			},
		),
		chromedp.WaitVisible(".main"),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.ActionFunc(func(ctx context.Context) error {

			log.Println("Open web page completed")

			for {
				select {
				case scene := <-ch:
					// create a timeout
					ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()

					err := chromedp.Evaluate(fmt.Sprintf(`PlayScene(%d);`, scene), &result).Do(ctxWithTimeout)
					if err != nil {
						if !strings.Contains(err.Error(), "encountered an undefined value") {
							return err
						}
					}
				default:
					time.Sleep(500 * time.Millisecond)
					err := callNetworkCGI()
					if err != nil {
						return err
					}
				}
			}
		}),
	)
	if err != nil {
		return err
	}

	return nil
}

// setHeadersAndNavigate returns a task list that sets the passed headers.
func setHeadersAndNavigate(host string, headers map[string]interface{}) chromedp.Tasks {
	return chromedp.Tasks{
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers(headers)),
		chromedp.Navigate(host),
	}
}

func callNetworkCGI() error {

	apiURL := os.Getenv("HOST")
	resource := "/network.cgi"

	data := url.Values{}
	data.Set("jsongetevent", "30")

	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String()

	client := &http.Client{}
	r, _ := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(data.Encode()))

	username := os.Getenv("USER")
	password := os.Getenv("PASS")
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))

	r.Header.Add("Authorization", authHeader)
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(r)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("http code not 200")
	}

	return nil
}
