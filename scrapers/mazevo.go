/*
	This file contains the code for the Mazevo scraper.
*/

package scrapers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/chromedp/cdproto/network"

	"github.com/chromedp/chromedp"
)

// ScrapeMazevo pulls Mazevo calendar events via the public API and stores the raw response.
func ScrapeMazevo(outDir string) {
	// Make output folder
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	ctx, cancel := utils.InitChromeDp()
	defer cancel()
	var reqID network.RequestID // ID for requests
	var eventsStart time.Time   // Start time of events request
	var eventsEnd time.Time     // End time of events request

	isPending := false
	isReceived := make(chan bool, 1)

	chromedp.ListenTarget(ctx, func(ev any) {
		switch ev := ev.(type) {
		case *network.EventRequestWillBeSent:
			// Get GetEvents Request Start and End Times (ISO 8601)
			if ev.Request.URL == "https://east.mymazevo.com/api/PublicCalendar/GetEvents" {
				rawPostData := ev.Request.PostDataEntries[0].Bytes
				decodedPostData, err := base64.StdEncoding.DecodeString(rawPostData)
				if err != nil {
					log.Panic(err)
				}
				var data map[string]any

				err = json.Unmarshal(decodedPostData, &data)
				if err != nil {
					log.Panic(err)
				}
				eventsStart, err = time.Parse(time.RFC3339, data["start"].(string))
				if err != nil {
					log.Panic(err)
				}
				eventsEnd, err = time.Parse(time.RFC3339, data["end"].(string))
				if err != nil {
					log.Panic(err)
				}

				// Check if end is 1 month after start
				if eventsEnd.After(eventsStart.AddDate(0, 1, -1)) {
					isPending = true
				}
			}
		case *network.EventResponseReceived:
			// Once Response is received, record the RequestID
			if isPending && ev.Response.URL == "https://east.mymazevo.com/api/PublicCalendar/GetEvents" {
				reqID = ev.RequestID
			}
		case *network.EventLoadingFinished:
			// Signal that response is finished loading
			if isPending && ev.RequestID == reqID {
				isPending = false
				isReceived <- true
			}
		}

	})
	_, err = chromedp.RunResponse(ctx,
		chromedp.Navigate("https://east.mymazevo.com/calendar/4219c6df695c03860350ea213837fe59"),
		chromedp.Sleep(5*time.Second),
		chromedp.Click("input#displayMonth", chromedp.NodeVisible),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// TODO: Account for error or network issue here
			// Wait until events have been received
			<-isReceived

			bodyBytes, err := network.GetResponseBody(reqID).Do(ctx)
			if err != nil {
				return fmt.Errorf("failed to get body: %w", err)
			}
			log.Printf("Scraped Mazevo from %s to %s!", eventsStart.Format(time.DateTime), eventsEnd.Format(time.DateTime))

			// Write event data to output file
			fptr, err := os.Create(fmt.Sprintf("%s/mazevoScraped.json", outDir))
			if err != nil {
				panic(err)
			}
			_, err = fptr.Write(bodyBytes)
			if err != nil {
				panic(err)
			}
			return nil
		}),
	)
	if err != nil {
		log.Panic(err)
	}

}
