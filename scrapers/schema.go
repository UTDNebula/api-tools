package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

type brochure struct {
	ProgramID int
	Content   string
}

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var brochureLinks []string
	var discoveryBody []byte
	var discoveryRequestID network.RequestID
	var programIDs []int

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
		programIDs = discovery.ReturnedPrograms
		fmt.Printf("programIDs=%v\n", programIDs)
	} else {
		programIDs = extractProgramIDs(brochureLinks)
		if len(programIDs) == 0 {
			log.Fatal("did not capture discovery-search response or any brochure links")
		}

		fmt.Printf("discovery response was not captured; using DOM fallback\n")
		fmt.Printf("programIDs=%v\n", programIDs)
	}

	for _, programID := range programIDs {
		brochure, err := fetchBrochure(programID)
		if err != nil {
			log.Printf("fetch brochure %d: %v", programID, err)
			continue
		}

		fmt.Printf("\nBROCHURE %d\n", brochure.ProgramID)
		fmt.Printf("content=%s\n", brochure.Content)
	}
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

func fetchBrochure(programID int) (brochure, error) {
	brochureURL := fmt.Sprintf(
		"https://utdallas-ea.terradotta.com/_portal/tds-program-brochure?programid=%d",
		programID,
	)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, brochureURL, nil)
	if err != nil {
		return brochure{}, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return brochure{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return brochure{}, fmt.Errorf("status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return brochure{}, err
	}

	return brochure{
		ProgramID: programID,
		Content:   string(body),
	}, nil
}
