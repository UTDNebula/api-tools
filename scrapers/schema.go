package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	discoveryURL      = "https://utdallas-ea.terradotta.com/_portal/program-discovery"
	discoveryEndpoint = "/program-api/program/discovery-search/"
)

type discoverySearchResponse struct {
	ReturnedPrograms []int   `json:"returnedPrograms"`
	ResultCount      int     `json:"resultCount"`
	MaxScore         float64 `json:"maxScore"`
	Start            int     `json:"start"`
	Limit            int     `json:"limit"`
	TotalCount       int     `json:"totalCount"`
	SearchID         string  `json:"searchid"`
}

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var brochureLinks []string
	var discoveryBody []byte
	var discoveryRequestID network.RequestID

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventResponseReceived:
			if strings.Contains(ev.Response.URL, discoveryEndpoint) {
				discoveryRequestID = ev.RequestID
			}
		}
	})

	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(discoveryURL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		// Insert filter/search actions here before waiting for results.
		chromedp.Sleep(8*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if discoveryRequestID == "" {
				return nil
			}

			var err error
			discoveryBody, err = network.GetResponseBody(discoveryRequestID).Do(ctx)
			return err
		}),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll("a"))
				.map(a => a.href)
				.filter(href => href.includes("tds-program-brochure"))
		`, &brochureLinks),
	)
	if err != nil {
		log.Fatal(err)
	}

	if len(discoveryBody) > 0 {
		var discovery discoverySearchResponse
		if err := json.Unmarshal(discoveryBody, &discovery); err != nil {
			log.Fatalf("parse discovery response: %v", err)
		}
		fmt.Printf("searchID=%s total=%d start=%d limit=%d returned=%d\n",
			discovery.SearchID,
			discovery.TotalCount,
			discovery.Start,
			discovery.Limit,
			discovery.ResultCount,
		)
		fmt.Printf("programIDs=%v\n", discovery.ReturnedPrograms)
		return
	}

	programIDs := extractProgramIDs(brochureLinks)
	if len(programIDs) == 0 {
		log.Fatal("did not capture discovery-search response or any brochure links")
	}

	fmt.Printf("discovery response was not captured; using DOM fallback\n")
	fmt.Printf("programIDs=%v\n", programIDs)
}

func extractProgramIDs(links []string) []int {
	ids := make([]int, 0, len(links))
	seen := make(map[int]struct{}, len(links))

	for _, rawLink := range links {
		parsed, err := url.Parse(rawLink)
		if err != nil {
			continue
		}

		programID, err := strconv.Atoi(parsed.Query().Get("programid"))
		if err != nil {
			continue
		}

		if _, ok := seen[programID]; ok {
			continue
		}

		seen[programID] = struct{}{}
		ids = append(ids, programID)
	}

	return ids
}
