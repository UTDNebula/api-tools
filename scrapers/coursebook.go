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
	prefixRegex = regexp.MustCompile("cp_[a-z]{0,5}")
	termRegex   = regexp.MustCompile("[0-9]{1,2}[sfu]")
)

const (
	reqThrottle    = 400 * time.Millisecond
	prefixThrottle = 5 * time.Second
	httpTimeout    = 10 * time.Second
)

// ScrapeCoursebook scrapes utd coursebook for the provided term (semester)
func ScrapeCoursebook(term string, startPrefix string, outDir string, resume bool) {
	if startPrefix != "" && !prefixRegex.MatchString(startPrefix) {
		log.Fatalf("Invalid starting prefix %s, must match format cp_{abcde}", startPrefix)
	}
	if !termRegex.MatchString(term) {
		log.Fatalf("Invalid term %s, must match format {00-99}{s/f/u}", term)
	}

	scraper := newCoursebookScraper(term, outDir)
	defer scraper.chromedpCancel()

	if resume && startPrefix == "" {
		// providing a starting prefix overrides the resume flag
		startPrefix = scraper.lastCompletePrefix()
	}

	log.Printf("[Begin Scrape] Starting scrape for term %s with %d prefixes", term, len(scraper.prefixes))

	totalTime := time.Now()
	for i, prefix := range scraper.prefixes {
		if startPrefix != "" && strings.Compare(prefix, startPrefix) < 0 {
			continue
		}

		start := time.Now()
		if err := scraper.ensurePrefixFolder(prefix); err != nil {
			log.Fatal(err)
		}

		var sectionIds []string
		var err error

		// if resume we skip existing entries otherwise overwrite them
		if resume {
			sectionIds, err = scraper.getMissingIdsForPrefix(prefix)
		} else {
			sectionIds, err = scraper.getSectionIdsForPrefix(prefix)
		}

		if err != nil {
			log.Fatalf("Error getting section ids for %s ", prefix)
		}

		if len(sectionIds) == 0 {
			log.Printf("No sections found for %s ", prefix)
			continue
		}

		log.Printf("[Scrape Prefix] %s (%d/%d): Found %d sections to scrape.", prefix, i+1, len(scraper.prefixes), len(sectionIds))

		for _, sectionId := range sectionIds {
			content, err := scraper.getSectionContent(sectionId)
			if err != nil {
				log.Fatalf("Error getting section content for section %s: %v", sectionId, err)
			}
			if err := scraper.writeSection(prefix, sectionId, content); err != nil {
				log.Fatalf("Error writing section %s: %v", sectionId, err)
			}
			time.Sleep(reqThrottle)
		}

		// At the end of the prefix loop
		log.Printf("[End Prefix] %s: Scraped %d sections in %v.", prefix, len(sectionIds), time.Since(start))
		time.Sleep(prefixThrottle)
	}
	log.Printf("[Scrape Complete] Finished scraping term %s in %v. Total sections %d: Total retries %d", term, time.Since(totalTime), scraper.totalScrapedSections, scraper.reqRetries)

	if err := scraper.validate(); err != nil {
		log.Fatal("Validating failed: ", err)
	}
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

func newCoursebookScraper(term string, outDir string) *coursebookScraper {
	ctx, cancel := utils.InitChromeDp()
	httpClient := &http.Client{
		Timeout: httpTimeout,
	}

	//prefixes in alphabetical order for skip prefix flag
	prefixes := utils.GetCoursePrefixes(ctx)
	sort.Strings(prefixes)
	return &coursebookScraper{
		chromedpCtx:       ctx,
		chromedpCancel:    cancel,
		httpClient:        httpClient,
		prefixes:          prefixes,
		coursebookHeaders: utils.RefreshToken(ctx),
		term:              term,
		outDir:            outDir,
		prefixIdsCache:    make(map[string][]string),
	}
}

// lastCompletePrefix returns the last prefix (alphabetical order) that contains
// html files for all of its section ids. returns an empty string if there are no
// complete prefixes
func (s *coursebookScraper) lastCompletePrefix() string {
	if err := s.ensureOutputFolder(); err != nil {
		log.Fatal(err)
	}

	dir, err := os.ReadDir(filepath.Join(s.outDir, s.term))
	if err != nil {
		log.Fatalf("failed to read output directory: %v", err)
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
			log.Fatalf("Failed to get ids: %v", err)
		}
		if len(missing) == 0 {
			return prefix
		}
		time.Sleep(reqThrottle)
	}
	return ""
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
		log.Panicf("Failed to access folder %s: %v", path, err)
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
			log.Fatalf("Http request failed: %v", err)
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
		s.coursebookHeaders = utils.RefreshToken(s.chromedpCtx)
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
			log.Fatal(err)
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
