/*
	This file contains the code for the coursebook scraper.
*/

package scrapers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/UTDNebula/api-tools/utils"
)

var (
	prefixRegex = regexp.MustCompile("^cp_[a-z]{1,5}$")
	termRegex   = regexp.MustCompile("[0-9]{1,2}[sfu]")
)

const (
	reqThrottle    = 400 * time.Millisecond
	prefixThrottle = 5 * time.Second
	httpTimeout    = 10 * time.Second
)

// ScrapeCoursebook Scrapes utd coursebook for provided term with specified options
func ScrapeCoursebook(term string, startPrefix string, outDir string, resume bool, retry int) {
	if term == "" {
		log.Fatal("Coursebook Scraping Setup Failed: No term specified for coursebook scraping! Use -term to specify.")
	}
	if startPrefix != "" && !prefixRegex.MatchString(startPrefix) {
		log.Fatalf("Coursebook Scraping Setup Failed: invalid starting prefix %s, must match format cp_{abcde}", startPrefix)
	}
	if !termRegex.MatchString(term) {
		log.Fatalf("Coursebook Scraping Setup Failed: invalid term %s, must match format {00-99}{s/f/u}", term)
	}

	var lastErr error = nil
	repeatErrCount := 0
	for repeatErrCount <= retry {
		err := func() error {
			scraper, err := newCoursebookScraper(term, outDir)
			if err != nil {
				return err
			}
			defer scraper.chromedpCancel()

			return scraper.Scrape(startPrefix, resume)
		}()

		// No error, scraped successfully
		if err == nil {
			return
		}

		// Context canceled Error (such as when closing chromedp window)
		if err.Error() == "context canceled" {
			log.Fatalf("Coursebook Scraping Canceled, Exiting")
		}

		/* Retry Coursebook Scraping */
		log.Printf("Coursebook Scraping Failed: %v", err)

		if fmt.Sprintf("%v", lastErr) == fmt.Sprintf("%v", err) {
			repeatErrCount++
		} else {
			repeatErrCount = 1
		}

		lastErr = err

		// TODO: handle netid (using setup error)
		// TODO: handle network issues -> wait longer before restarting
		// TODO: ensure all panics are reasonable, and should not be retried
	}

	if retry != 0 {
		log.Fatalf("Coursebook Scraping Failed %d times in a row with the same error, Exiting", retry+1)
	}
}

// Scrape begins the scraping process for all prefixes
func (s *coursebookScraper) Scrape(startPrefix string, resume bool) error {
	if resume && startPrefix == "" {
		// providing a starting prefix overrides the resume flag
		var err error
		startPrefix, err = s.lastCompletePrefix()
		if err != nil {
			return fmt.Errorf("failed to get last complete prefix while resuming: %v", err)
		}
	}

	log.Printf("[Begin Scrape] Starting scrape for term %s with %d prefixes", s.term, len(s.prefixes))

	totalTime := time.Now()
	for i, prefix := range s.prefixes {
		if startPrefix != "" && strings.Compare(prefix, startPrefix) < 0 {
			continue
		}

		if err := s.scrapePrefix(prefix, resume, i); err != nil {
			return err
		}
	}
	log.Printf("[Scrape Complete] Finished scraping term %s in %v. Total sections %d: Total retries %d", s.term, time.Since(totalTime), s.totalScrapedSections, s.reqRetries)

	if err := s.validate(); err != nil {
		log.Panicf("Validating failed: %v", err)
	}

	return nil
}

// scrapePrefix scrapes all sections for a single prefix
func (s *coursebookScraper) scrapePrefix(prefix string, resume bool, index int) error {
	start := time.Now()
	if err := s.ensurePrefixFolder(prefix); err != nil {
		log.Panic(err)
	}

	var sectionIds []string
	var err error

	// if resume we skip existing entries otherwise overwrite them
	if resume {
		sectionIds, err = s.getMissingIdsForPrefix(prefix)
	} else {
		sectionIds, err = s.getSectionIdsForPrefix(prefix)
	}

	if err != nil {
		log.Panicf("Error getting section ids for %s ", prefix)
	}

	if len(sectionIds) == 0 {
		log.Printf("No sections found for %s ", prefix)
		return nil
	}

	log.Printf("[Scrape Prefix] %s (%d/%d): Found %d sections to scrape.", prefix, index+1, len(s.prefixes), len(sectionIds))

	for _, sectionId := range sectionIds {
		content, err := s.getSectionContent(sectionId) // TODO: This function can have the following, dont force chromedp restart when it happens
		// error getting section content for section angm3305.0w1.26u: get section content for id angm3305.0w1.26u failed: Post "https://coursebook.utdallas.edu/clips/clip-cb11-hat.zog": context deadline exceeded (Client.Timeout exceeded while awaiting headers)

		if err != nil {
			return fmt.Errorf("error getting section content for section %s: %v", sectionId, err)
		}
		if err := s.writeSection(prefix, sectionId, content); err != nil {
			log.Panicf("Error writing section %s: %v", sectionId, err)
		}
		time.Sleep(reqThrottle)
	}

	// At the end of the prefix loop
	log.Printf("[End Prefix] %s: Scraped %d sections in %v.", prefix, len(sectionIds), time.Since(start))
	time.Sleep(prefixThrottle)
	return nil
}

type coursebookScraper struct {
	chromedpCtx       context.Context
	chromedpCancel    context.CancelFunc
	httpClient        *http.Client
	coursebookHeaders map[string][]string
	prefixes          []string
	term              string
	outDir            string

	prefixIdsCache map[string][]string

	//metrics
	reqRetries           int
	totalScrapedSections int
}

func newCoursebookScraper(term string, outDir string) (*coursebookScraper, error) {
	ctx, cancel := utils.InitChromeDp()
	httpClient := &http.Client{
		Timeout: httpTimeout,
	}

	//prefixes in alphabetical order for skip prefix flag
	prefixes, err := utils.GetCoursePrefixes(ctx)
	if err != nil {
		return nil, err
	}
	sort.Strings(prefixes)
	coursebookHeaders, err := utils.RefreshToken(ctx)
	if err != nil {
		return nil, err
	}
	return &coursebookScraper{
		chromedpCtx:       ctx,
		chromedpCancel:    cancel,
		httpClient:        httpClient,
		prefixes:          prefixes,
		coursebookHeaders: coursebookHeaders,
		term:              term,
		outDir:            outDir,
		prefixIdsCache:    make(map[string][]string),
	}, nil
}

// lastCompletePrefix returns the last prefix (alphabetical order) that contains
// html files for all of its section ids. returns an empty string if there are no
// complete prefixes
func (s *coursebookScraper) lastCompletePrefix() (string, error) {
	if err := s.ensureOutputFolder(); err != nil {
		return "", err
	}

	dir, err := os.ReadDir(filepath.Join(s.outDir, s.term))
	if err != nil {
		return "", fmt.Errorf("failed to read output directory: %w", err)
	}

	foundPrefixes := make([]string, 0, len(s.prefixes))
	for _, file := range dir {
		foundPrefixes = append(foundPrefixes, file.Name())
	}

	sort.Strings(foundPrefixes)
	slices.Reverse(foundPrefixes)

	for _, prefix := range foundPrefixes {
		missing, err := s.getMissingIdsForPrefix(prefix)
		if err != nil {
			return "", fmt.Errorf("failed to get ids: %w", err)
		}
		if len(missing) == 0 {
			return prefix, nil
		}
		time.Sleep(reqThrottle)
	}
	return "", nil
}

// ensurePrefixFolder creates {outDir}/term if it does not exist
func (s *coursebookScraper) ensureOutputFolder() error {
	if err := os.MkdirAll(filepath.Join(s.outDir, s.term), 0755); err != nil {
		return fmt.Errorf("failed to create term forlder: %w", err)
	}
	return nil
}

// ensurePrefixFolder creates {outDir}/term/prefix if it does not exist
func (s *coursebookScraper) ensurePrefixFolder(prefix string) error {
	if err := os.MkdirAll(filepath.Join(s.outDir, s.term, prefix), 0755); err != nil {
		return fmt.Errorf("failed to create folder for %s: %w", prefix, err)
	}
	return nil
}

// writeSection writes content to file {outDir}/term/prefix/{id}.html
func (s *coursebookScraper) writeSection(prefix string, id string, content string) error {
	if err := os.WriteFile(filepath.Join(s.outDir, s.term, prefix, id+".html"), []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write section %s: %w", id, err)
	}
	return nil
}

// getSectionContent calls internal coursebook api to get the html for the provided id
// retries up to 3 times, each time refreshing the token and waiting longer
func (s *coursebookScraper) getSectionContent(id string) (string, error) {
	queryStr := fmt.Sprintf("id=%s&req=b30da8ab21637dbef35fd7682f48e1c1W0ypMhaj%%2FdsnYn3Wa03BrxSNgCeyvLfvucSTobcSXRf38SWaUaNfMjJQn%%2BdcabF%%2F7ZuG%%2BdKqHAqmrxEKyg8AdB0FqVGcz4rkff3%%2B3SIUIt8%%3D&action=info", id)
	response, err := s.req(queryStr, 3, id)
	if err != nil {
		return "", fmt.Errorf("get section content for id %s failed: %w", id, err)
	}
	s.totalScrapedSections++
	return response, nil
}

// getMissingIdsForPrefix calls getSectionIdsForPrefix and filters out the ids that already
// exist in the prefix directory
func (s *coursebookScraper) getMissingIdsForPrefix(prefix string) ([]string, error) {
	path := filepath.Join(s.outDir, s.term, prefix)

	sectionIds, err := s.getSectionIdsForPrefix(prefix)
	if err != nil {
		return sectionIds, err
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return sectionIds, nil
		}
		return sectionIds, fmt.Errorf("failed to access folder %s: %w", path, err)
	}

	dir, err := os.ReadDir(path)
	if err != nil {
		return sectionIds, fmt.Errorf("failed to access folder %s: %w", path, err)
	}

	foundIds := make(map[string]bool)
	for _, file := range dir {
		id := strings.TrimSuffix(file.Name(), ".html")
		foundIds[id] = true
	}

	var filteredIds []string
	for _, id := range sectionIds {
		if !foundIds[id] {
			filteredIds = append(filteredIds, id)
		}
	}

	return filteredIds, nil
}

// getSectionIdsForPrefix calls internal coursebook api to get all section ids for the provide prefix
// retries up to 10 times, each time refreshing the token and waiting longer.
func (s *coursebookScraper) getSectionIdsForPrefix(prefix string) ([]string, error) {
	if ids, ok := s.prefixIdsCache[prefix]; ok {
		return ids, nil
	}

	sections := make([]string, 0, 100)
	for _, clevel := range []string{"clevel_u", "clevel_g"} {
		queryStr := fmt.Sprintf("action=search&s%%5B%%5D=term_%s&s%%5B%%5D=%s&s%%5B%%5D=%s", s.term, prefix, clevel)
		content, err := s.req(queryStr, 10, fmt.Sprintf("%s:%s", prefix, clevel))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch sections: %s", err)
		}
		sectionRegexp := utils.Regexpf(`View details for section (%s%s\.\w+\.%s)`, prefix[3:], utils.R_COURSE_CODE, utils.R_TERM_CODE)
		matches := sectionRegexp.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			sections = append(sections, match[1])
		}
	}

	s.prefixIdsCache[prefix] = sections
	return sections, nil
}

// req utility function for making calling the coursebook api
func (s *coursebookScraper) req(queryStr string, retries int, reqName string) (string, error) {
	var res *http.Response
	err := utils.Retry(func() error {
		req, err := http.NewRequest("POST", "https://coursebook.utdallas.edu/clips/clip-cb11-hat.zog", strings.NewReader(queryStr))
		if err != nil {
			return fmt.Errorf("http request failed: %w", err)
		}
		req.Header = s.coursebookHeaders

		start := time.Now()
		res, err = s.httpClient.Do(req)
		dur := time.Since(start)

		if res != nil {
			if res.StatusCode != 200 {
				return errors.New("non-200 response status code")
			}
			utils.VPrintf("[Request Success] Request for [%s] took %v", reqName, dur)
		} else if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				utils.VPrintf("[Timeout] Request for [%s] timed out", reqName)
			} else {
				utils.VPrintf("[Request Error] Request for %s failed: %v", reqName, err)
			}
		}

		return err
	}, retries, func(numRetries int) {
		utils.VPrintf("[Request Retry] Attempt %d of %d for request %s", numRetries, retries, reqName)
		coursebookHeaders, err := utils.RefreshToken(s.chromedpCtx)
		if err != nil {
			utils.VPrintf("[Token Refresh Failed] Failed to refresh token during retry for request %s: %v", reqName, err)
		} else {
			s.coursebookHeaders = coursebookHeaders
		}

		s.reqRetries++

		//back off exponentially
		time.Sleep(time.Duration(math.Pow(2, float64(numRetries))) * time.Second)
	})
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	content, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %s", err)
	}
	return string(content), nil
}

// validate returns true if each prefix contains all required ids
// if it does not it will re-scrape all missing sections
func (s *coursebookScraper) validate() error {
	log.Printf("[Begin Validation] Starting Validation for term %s", s.term)

	for _, prefix := range s.prefixes {
		ids, err := s.getMissingIdsForPrefix(prefix)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			log.Printf("[Validation] %s is correct", prefix)
			continue
		}

		log.Printf("[Validation] Missing %d sections for %s", len(ids), prefix)

		if err := s.ensurePrefixFolder(prefix); err != nil {
			log.Panic(err)
		}

		for _, id := range ids {
			content, err := s.getSectionContent(id)
			if err != nil {
				return fmt.Errorf("error getting section content for section %s: %v", id, err)
			}
			if err := s.writeSection(prefix, id, content); err != nil {
				return fmt.Errorf("error writing section %s: %v", id, err)
			}
			time.Sleep(reqThrottle)
		}

		log.Printf("[Validation] %s is correct", prefix)
		time.Sleep(prefixThrottle)
	}

	log.Print("[End Validation] Validation Successful")
	return nil
}
