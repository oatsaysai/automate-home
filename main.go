package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

var ch = make(chan int64)

func main() {

	go openWebPage(ch)

	http.HandleFunc("/play-scene", playSceneHandler)

	fmt.Println("Server started at http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
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

func openWebPage(ch chan int64) {

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
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
			for {
				scene := <-ch
				chromedp.Evaluate(fmt.Sprintf(`PlayScene(%d);`, scene), &result).Do(ctx)
			}
		}),
	)
	if err != nil {
		log.Fatalln(err)
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
