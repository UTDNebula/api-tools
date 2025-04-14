/*
	This file contains the code for the coursebook scraper.
*/

package scrapers

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/UTDNebula/api-tools/utils"
)

func ScrapeCoursebook(term string, startPrefix string, outDir string) {
	// Start chromedp
	chromedpCtx, cancel := utils.InitChromeDp()
	defer cancel()

	coursePrefixes := utils.GetCoursePrefixes(chromedpCtx)

	// Find index of starting prefix, if one has been given
	startPrefixIndex := 0
	if startPrefix != "" && startPrefix != coursePrefixes[0] {
		for i, prefix := range coursePrefixes {
			if prefix == startPrefix {
				startPrefixIndex = i
				break
			}
		}
		if startPrefixIndex == 0 {
			log.Panic("Failed to find provided course prefix! Remember, the format is cp_<PREFIX>!")
		}
	}

	// Init http client
	cli := &http.Client{}

	// Make the output directory for this term
	termDir := fmt.Sprintf("%s/%s", outDir, term)
	if err := os.MkdirAll(termDir, 0777); err != nil {
		panic(err)
	}

	// Keep track of how many total sections we've scraped
	totalSections := 0

	// Scrape all sections for each course prefix
	for prefixIndex, coursePrefix := range coursePrefixes {

		// Skip to startPrefixIndex
		if prefixIndex < startPrefixIndex {
			continue
		}

		// Make a directory in the output for this course prefix
		courseDir := fmt.Sprintf("%s/%s", termDir, coursePrefix)
		if err := os.MkdirAll(courseDir, 0777); err != nil {
			panic(err)
		}
		// Get a fresh token at the start of each new prefix because we can lol
		coursebookHeaders := utils.RefreshToken(chromedpCtx)
		// Give coursebook some time to recognize the new token
		time.Sleep(500 * time.Millisecond)
		// String builder to store accumulated course HTML data for both class levels
		courseBuilder := strings.Builder{}

		log.Printf("Finding sections for course prefix %s...", coursePrefix)

		// Get courses for term and prefix, split by grad and undergrad to avoid 300 section cap
		for _, clevel := range []string{"clevel_u", "clevel_g"} {
			queryStr := fmt.Sprintf("action=search&s%%5B%%5D=term_%s&s%%5B%%5D=%s&s%%5B%%5D=%s", term, coursePrefix, clevel)

			// Try HTTP request, retrying if necessary
			var res *http.Response
			err := utils.Retry(func() error {
				req, err := http.NewRequest("POST", "https://coursebook.utdallas.edu/clips/clip-cb11-hat.zog", strings.NewReader(queryStr))
				if err != nil {
					panic(err)
				}
				req.Header = coursebookHeaders
				res, err = cli.Do(req)
				if res.StatusCode != 200 {
					return errors.New("Non-200 response status code")
				}
				return err
			}, 10, func(numRetries int) {
				log.Printf("WARN: Section find for course prefix %s failed! Performing retry #%d...", coursePrefix, numRetries)
				coursebookHeaders = utils.RefreshToken(chromedpCtx)
				// Wait proportionally long to how many times we've retried; generally works pretty well
				time.Sleep(500 * time.Millisecond * time.Duration(numRetries))
			})

			if err != nil {
				panic(err)
			}

			buf := bytes.Buffer{}
			buf.ReadFrom(res.Body)
			courseBuilder.Write(buf.Bytes())
		}
		// Find all section IDs in returned data
		sectionRegexp := utils.Regexpf(`View details for section (%s%s\.\w+\.%s)`, coursePrefix[3:], utils.R_COURSE_CODE, utils.R_TERM_CODE)
		smatches := sectionRegexp.FindAllStringSubmatch(courseBuilder.String(), -1)
		sectionIDs := make([]string, 0, len(smatches))
		for _, matchSet := range smatches {
			sectionIDs = append(sectionIDs, matchSet[1])
		}
		log.Printf("Found %d sections for course prefix %s", len(sectionIDs), coursePrefix)

		// Get a new token before starting the section lookup
		coursebookHeaders = utils.RefreshToken(chromedpCtx)
		// Give coursebook some time to recognize the new token
		time.Sleep(500 * time.Millisecond)

		// Get HTML data for all section IDs
		sectionsInCoursePrefix := 0
		for sectionIndex, id := range sectionIDs {

			// Get section info
			// Worth noting that the "req" param in the request below doesn't actually seem to matter... consider it filler to make sure the request goes through
			queryStr := fmt.Sprintf("id=%s&req=b30da8ab21637dbef35fd7682f48e1c1W0ypMhaj%%2FdsnYn3Wa03BrxSNgCeyvLfvucSTobcSXRf38SWaUaNfMjJQn%%2BdcabF%%2F7ZuG%%2BdKqHAqmrxEKyg8AdB0FqVGcz4rkff3%%2B3SIUIt8%%3D&action=info", id)

			// Try HTTP request, retrying if necessary
			var res *http.Response
			err := utils.Retry(func() error {
				req, err := http.NewRequest("POST", "https://coursebook.utdallas.edu/clips/clip-cb11-hat.zog", strings.NewReader(queryStr))
				if err != nil {
					panic(err)
				}
				req.Header = coursebookHeaders
				res, err = cli.Do(req)
				if res.StatusCode != 200 {
					return errors.New("Non-200 response status code")
				}
				return err
			}, 10, func(numRetries int) {
				log.Printf("WARN: Section id lookup for id %s failed! Performing retry #%d...", id, numRetries)
				coursebookHeaders = utils.RefreshToken(chromedpCtx)
				// Wait proportionally long to how many times we've retried; generally works pretty well
				time.Sleep(500 * time.Millisecond * time.Duration(numRetries))
			})

			if err != nil {
				panic(err)
			}

			fptr, err := os.Create(fmt.Sprintf("%s/%s.html", courseDir, id))
			if err != nil {
				panic(err)
			}
			buf := bytes.Buffer{}
			buf.ReadFrom(res.Body)
			if _, err := fptr.Write(buf.Bytes()); err != nil {
				panic(err)
			}
			fptr.Close()

			// Report success, refresh token periodically
			utils.VPrintf("Got section: %s", id)
			if sectionIndex%30 == 0 && sectionIndex != 0 {
				// Ratelimit? What ratelimit?
				coursebookHeaders = utils.RefreshToken(chromedpCtx)
				// Give coursebook some time to recognize the new token
				time.Sleep(500 * time.Millisecond)
			}
			sectionsInCoursePrefix++
		}
		log.Printf("Finished scraping course prefix %s. Got %d sections.\n----------------------------------------------------", coursePrefix, sectionsInCoursePrefix)
		// Panic if we got fewer sections than we should've
		if sectionsInCoursePrefix != len(sectionIDs) {
			log.Panicf("Section count mismatch! Expected sections %d for %s, got %d", sectionsInCoursePrefix, coursePrefix, sectionsInCoursePrefix)
		}
		totalSections += sectionsInCoursePrefix
	}
	log.Printf("Done scraping term! Scraped %d sections.", totalSections)

}
