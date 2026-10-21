/*
	This file contains the code for the study abroad program scraper.
*/

package scrapers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	discoveryURL       = "https://utdallas-ea.terradotta.com/_portal/program-discovery"
	discoveryEndpoint  = "/program-api/program/discovery-search/"
	discoverySearchURL = "https://us-prod-api-v2.terradotta.com/program-api/program/discovery-search/"
	brochureURLFormat  = "https://utdallas-ea.terradotta.com/models/services/REST/index.cfm?endpoint=/v3/program/%d/brochure"

	// discoveryPageSize is the number of programs requested per discovery-search page.
	// The portal itself uses 12; a larger page just means fewer round trips.
	discoveryPageSize = 100
)

// discoverySearchRequest is the JSON body the discovery portal POSTs to discovery-search.
// Only active (publicly visible) programs are requested, matching the portal's default filter.
type discoverySearchRequest struct {
	Filter        []discoveryFilter `json:"filter"`
	Limit         int               `json:"limit"`
	SortBy        string            `json:"sortBy"`
	SortDirection string            `json:"sortDirection"`
	Start         int               `json:"start"`
}

type discoveryFilter struct {
	FilterType   string                 `json:"filterType"`
	FilterValues []discoveryFilterValue `json:"filterValues"`
}

type discoveryFilterValue struct {
	Value bool `json:"value"`
}

type discoverySearchResponse struct {
	ReturnedPrograms []int `json:"returnedPrograms"`
	// Records are kept raw so the parser sees everything discovery returns.
	// They carry the program name, terms, and locations, which the brochure endpoint does not.
	Records     []json.RawMessage `json:"records"`
	ResultCount int               `json:"resultCount"`
	MaxScore    float64           `json:"maxScore"`
	Start       int               `json:"start"`
	Limit       int               `json:"limit"`
	TotalCount  int               `json:"totalCount"`
	SearchID    string            `json:"searchid"`
}

// ScrapeStudyAbroad saves every active study abroad program brochure, plus the discovery search
// records that describe them, to {outDir}/studyAbroad for the parser to combine.
// Set PROGRAM_IDS to a comma-separated list to scrape specific programs instead of discovering them all.
func ScrapeStudyAbroad(outDir string) {
	programIDs, err := programIDsFromEnv(os.Getenv("PROGRAM_IDS"))
	if err != nil {
		log.Panic(err)
	}

	studyAbroadDir := filepath.Join(outDir, "studyAbroad")
	brochureDir := filepath.Join(studyAbroadDir, "brochures")

	if len(programIDs) == 0 {
		var records []json.RawMessage
		programIDs, records, err = discoverPrograms()
		if err != nil {
			log.Panic(err)
		}

		// A full scrape replaces the previous one, so brochures for programs that are no longer active don't linger.
		if err := os.RemoveAll(brochureDir); err != nil {
			log.Panic(err)
		}
		if err := os.MkdirAll(studyAbroadDir, 0777); err != nil {
			log.Panic(err)
		}
		if err := utils.WriteJSON(filepath.Join(studyAbroadDir, "discoveryScraped.json"), records); err != nil {
			log.Panic(err)
		}
	} else {
		log.Printf("Using PROGRAM_IDS=%v", programIDs)
	}

	if err := os.MkdirAll(brochureDir, 0777); err != nil {
		log.Panic(err)
	}

	log.Printf("Fetching %d program brochures...", len(programIDs))

	scraped := 0
	for _, programID := range programIDs {
		content, err := fetchBrochure(programID)
		if err != nil {
			log.Printf("fetch brochure %d: %v", programID, err)
			continue
		}

		if err := os.WriteFile(filepath.Join(brochureDir, fmt.Sprintf("%d.json", programID)), content, 0644); err != nil {
			log.Printf("write brochure %d: %v", programID, err)
			continue
		}
		scraped++
	}

	log.Printf("Scraped %d/%d study abroad programs to %s!", scraped, len(programIDs), studyAbroadDir)
}

// discoverPrograms loads the discovery portal to capture an API token, then pages through
// discovery-search to collect every active program ID along with its raw discovery record.
func discoverPrograms() ([]int, []json.RawMessage, error) {
	authorization, err := captureDiscoveryToken()
	if err != nil {
		return nil, nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}

	programIDs := make([]int, 0)
	records := make([]json.RawMessage, 0)
	seen := make(map[int]struct{})

	for start := 0; ; start += discoveryPageSize {
		page, err := fetchDiscoveryPage(client, authorization, start)
		if err != nil {
			return nil, nil, err
		}

		if start == 0 {
			log.Printf("Found %d active study abroad programs (searchID=%s)", page.TotalCount, page.SearchID)
		}

		records = append(records, page.Records...)

		for _, programID := range page.ReturnedPrograms {
			if _, ok := seen[programID]; ok {
				continue
			}
			seen[programID] = struct{}{}
			programIDs = append(programIDs, programID)
		}

		// Stop once the API returns an empty page or we've collected everything it reported.
		if page.ResultCount == 0 || len(programIDs) >= page.TotalCount {
			break
		}
	}

	return programIDs, records, nil
}

// captureDiscoveryToken loads the discovery portal in a browser and intercepts the
// Authorization header of the discovery-search request the page makes on its own.
func captureDiscoveryToken() (string, error) {
	browserCtx, browserCancel := utils.InitChromeDp()
	defer browserCancel()

	ctx, cancel := context.WithTimeout(browserCtx, 60*time.Second)
	defer cancel()

	var mutex sync.Mutex
	var token string
	captured := make(chan struct{})

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		request, ok := ev.(*network.EventRequestWillBeSent)
		if !ok || !strings.Contains(request.Request.URL, discoveryEndpoint) {
			return
		}

		authorization, ok := request.Request.Headers["Authorization"].(string)
		if !ok || authorization == "" {
			return
		}

		mutex.Lock()
		defer mutex.Unlock()
		if token == "" {
			token = authorization
			close(captured)
		}
	})

	log.Println("Loading program discovery portal...")
	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(discoveryURL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			select {
			case <-captured:
				return nil
			case <-ctx.Done():
				return fmt.Errorf("timed out waiting for a discovery-search request")
			}
		}),
	)
	if err != nil {
		return "", err
	}

	mutex.Lock()
	defer mutex.Unlock()
	if token == "" {
		return "", fmt.Errorf("did not capture a discovery-search authorization token")
	}

	return token, nil
}

func fetchDiscoveryPage(client *http.Client, authorization string, start int) (discoverySearchResponse, error) {
	body, err := json.Marshal(discoverySearchRequest{
		Filter: []discoveryFilter{{
			FilterType:   "programDiscoveryIsActive",
			FilterValues: []discoveryFilterValue{{Value: true}},
		}},
		Limit:         discoveryPageSize,
		SortBy:        "_score",
		SortDirection: "DESC",
		Start:         start,
	})
	if err != nil {
		return discoverySearchResponse{}, err
	}

	req, err := http.NewRequest(http.MethodPost, discoverySearchURL, bytes.NewReader(body))
	if err != nil {
		return discoverySearchResponse{}, err
	}

	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Origin", "https://utdallas-ea.terradotta.com")
	req.Header.Set("Referer", "https://utdallas-ea.terradotta.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return discoverySearchResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return discoverySearchResponse{}, fmt.Errorf("discovery-search start=%d: status %s: %s", start, resp.Status, strings.TrimSpace(string(preview)))
	}

	var page discoverySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return discoverySearchResponse{}, fmt.Errorf("parse discovery response: %w", err)
	}

	return page, nil
}

func programIDsFromEnv(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	ids := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		programID, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("parse PROGRAM_IDS value %q: %w", part, err)
		}

		ids = append(ids, programID)
	}

	return ids, nil
}

func fetchBrochure(programID int) ([]byte, error) {
	brochureURL := fmt.Sprintf(brochureURLFormat, programID)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, brochureURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return io.ReadAll(resp.Body)
}
