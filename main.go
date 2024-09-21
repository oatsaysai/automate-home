package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func main() {

	// Get ENV
	cmdVal := os.Getenv("CMD")
	sceneVal := os.Getenv("SCENE")

	if cmdVal == "serve" {
		http.HandleFunc("/play-scene", playSceneHandler)

		fmt.Println("Server started at http://localhost:8080")
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			fmt.Println("Error starting server:", err)
		}
	} else if cmdVal == "playScene" {
		i, err := strconv.ParseInt(sceneVal, 10, 64)
		if err != nil {
			panic(err)
		}

		playScene(i)
	}
}

type PlaySceneParams struct {
	Scene int64 `json:"scene"`
}

func errorHandler() {
	if r := recover(); r != nil {
		log.Printf("Recovered from panic: %v", r)
	}
}

func playSceneHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler()

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

	cmd := exec.Command("./automate-home")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "CMD=playScene")
	cmd.Env = append(cmd.Env, fmt.Sprintf("SCENE=%d", params.Scene))
	err = cmd.Run()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Fprintf(w, "Play scene: %+v\n", params.Scene)
}

func playScene(scene int64) {

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// url := "http://192.168.1.6"
	url := ""

	username := ""
	password := ""
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
		chromedp.Evaluate(fmt.Sprintf(`PlayScene(%d);`, scene), &result),
	)
	if err != nil {
		log.Fatal(err)
	}
}

// setHeadersAndNavigate returns a task list that sets the passed headers.
func setHeadersAndNavigate(host string, headers map[string]interface{}) chromedp.Tasks {
	return chromedp.Tasks{
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers(headers)),
		chromedp.Navigate(host),
	}
}
