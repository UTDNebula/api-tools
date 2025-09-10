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
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/UTDNebula/api-tools/utils"
)

// ScrapeCoursebook scrapes utd coursebook for the provided term (semester)
// if resume flag is true then
func ScrapeCoursebook(term string, startPrefix string, outDir string, resume bool) {
	scraper := newCoursebookScraper(term, outDir)
	defer scraper.cancel()

	if startPrefix == "" {
		// providing a starting prefix overrides the resume flag
		startPrefix = scraper.lastCompletePrefix()
	}

	skipped := 0
	for _, prefix := range scraper.prefixes {
		if startPrefix == "" || strings.Compare(prefix, startPrefix) == -1 {
			utils.VPrintf("Skipping prefix %s", prefix)
			continue
		}

		if skipped != -1 {
			log.Printf("Skipped %d prefixes", skipped)
			skipped = -1
		}

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
			log.Fatal("Error getting section ids for prefix ", prefix)
		}

		for _, sectionId := range sectionIds {
			content, err := scraper.getSectionContent(sectionId)
			if err != nil {
				log.Fatalf("Error getting section content for section %s: %v", sectionId, err)
			}
			if err := scraper.writeSection(prefix, sectionId, content); err != nil {
				log.Fatalf("Error writing section %s: %v", sectionId, err)
			}
			utils.VPrintf("Got section: %s", sectionId)

			// wait 3 seconds between requests to avoid rate limiting
			time.Sleep(3 * time.Second)
		}

	}
	log.Print("Done scraping term!")
	log.Print("Validating... ")

	success, err := scraper.validate()
	if err != nil {
		log.Fatal("Validating failed: ", err)
	} else if success {
		log.Print("Validating successful!")
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
}

func newCoursebookScraper(term string, outDir string) *coursebookScraper {
	ctx, cancel := utils.InitChromeDp()
	httpClient := &http.Client{}

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
	}
}

// lastCompletePrefix returns the last prefix (alphabetical order) that contains
// html files for all of its section ids. returns an empty string if there are no
// complete prefixes
func (s *coursebookScraper) lastCompletePrefix() string {
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
	}
	return ""
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
	response, err := s.req(queryStr, 3, fmt.Sprintf("section %s content", id))
	if err != nil {
		return "", fmt.Errorf("get section content for id %s failed: %w", id, err)
	}
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
		} else {
			utils.VPrintf("Found section: %s", id)

		}
	}

	return filteredIds, nil
}

// getSectionIdsForPrefix calls internal coursebook api to get all section ids for the provide prefix
// retries up to 10 times, each time refreshing the token and waiting longer
func (s *coursebookScraper) getSectionIdsForPrefix(prefix string) ([]string, error) {
	sections := make([]string, 0, 100)
	for _, clevel := range []string{"clevel_u", "clevel_g"} {
		queryStr := fmt.Sprintf("action=search&s%%5B%%5D=term_%s&s%%5B%%5D=%s&s%%5B%%5D=%s", s.term, prefix, clevel)
		content, err := s.req(queryStr, 10, fmt.Sprintf("sections for %s", prefix))
		if err != nil {
			return []string{}, fmt.Errorf("failed to fetch sections: %s", err)
		}

		sectionRegexp := utils.Regexpf(`View details for section (%s%s\.\w+\.%s)`, prefix[3:], utils.R_COURSE_CODE, utils.R_TERM_CODE)
		smatches := sectionRegexp.FindAllStringSubmatch(content, -1)
		for _, match := range smatches {
			sections = append(sections, match[1])
		}
	}
	log.Printf("Found %d sections for %s", len(sections), prefix)
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
		res, err = s.httpClient.Do(req)
		if res != nil && res.StatusCode != 200 {
			return errors.New("non-200 response status code")
		}
		return err
	}, retries, func(numRetries int) {
		log.Printf("Failed to get %s, Retry %d of %d", reqName, numRetries, retries)
		s.coursebookHeaders = utils.RefreshToken(s.chromedpCtx)

		// front load delay since if the first one fails it is likely the next few will as well
		time.Sleep((8 * time.Second) + (500 * time.Millisecond * time.Duration(numRetries)))
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

// refreshToken token using login info
func (s *coursebookScraper) refreshToken() {
	s.coursebookHeaders = utils.RefreshToken(s.chromedpCtx)
}

// cancel cancels chromedp context
func (s *coursebookScraper) cancel() {
	s.chromedpCancel()
}

// validate returns true if each prefix contains all required ids
func (s *coursebookScraper) validate() (bool, error) {
	missing := make(map[string][]string)

	count := 0
	for _, prefix := range s.prefixes {
		ids, err := s.getMissingIdsForPrefix(prefix)
		if err != nil {
			return false, err
		}
		if len(ids) > 0 {
			count++
		}
		missing[prefix] = ids
		time.Sleep(5 * time.Second)
	}

	for prefix, ids := range missing {
		if len(ids) > 0 {
			log.Printf("Missing %d sections for prefix: %s", len(ids), prefix)

			for _, id := range ids {
				content, err := s.getSectionContent(id)
				if err != nil {
					return false, fmt.Errorf("error getting section content for section %s: %v", id, err)
				}
				if err := s.writeSection(prefix, id, content); err != nil {
					return false, fmt.Errorf("error writing section %s: %v", id, err)
				}
				utils.VPrintf("Got section: %s", id)
				time.Sleep(3 * time.Second)
			}
		}
	}
	return true, nil
}
