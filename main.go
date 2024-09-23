package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
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
			time.Sleep(3000 * time.Millisecond)
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

	log.Printf("Play scene: %+v\n", params.Scene)
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

	js := fmt.Sprintf(
		`
		var details = {
			'jsongetevent': '173'
		};

		var formBody = [];
		for (var property in details) {
		var encodedKey = encodeURIComponent(property);
		var encodedValue = encodeURIComponent(details[property]);
		formBody.push(encodedKey + "=" + encodedValue);
		}
		formBody = formBody.join("&");

		(async function() {
			const response = await fetch('%s', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8',
					'Authorization': '%s'
				},
				body: formBody
			});
			const data = await response.text();
			return data;
		})()
		`,
		url+"/network.cgi",
		authHeader,
	)

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

			chForBreak := make(chan error)

			go func() {
				for {
					// create a timeout
					time.Sleep(5000 * time.Millisecond)

					ctxWithTimeout, _ := context.WithTimeout(ctx, 5*time.Second)

					err := chromedp.Evaluate(js, &result, func(ep *runtime.EvaluateParams) *runtime.EvaluateParams {
						return ep.WithAwaitPromise(true)
					}).Do(ctxWithTimeout)
					if err != nil {
						chForBreak <- err
					}
					if !strings.Contains(result, "jsongetevent done") {
						chForBreak <- errors.New("got error from gateway")
					}
				}
			}()

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
				case err := <-chForBreak:
					return err
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
