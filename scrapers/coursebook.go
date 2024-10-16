/*
	This file contains the code for the coursebook scraper.
*/

package scrapers

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/joho/godotenv"
)

func ScrapeCoursebook(term string, startPrefix string, outDir string) {

	// Load env vars
	if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
	}

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
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	cli := &http.Client{Transport: tr}

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
			res, err := utils.RetryHTTP(func() *http.Request {
				req, err := http.NewRequest("POST", "https://coursebook.utdallas.edu/clips/clip-cb11-hat.zog", strings.NewReader(queryStr))
				if err != nil {
					panic(err)
				}
				req.Header = coursebookHeaders
				return req
			}, cli, func(res *http.Response, numRetries int) {
				log.Printf("ERROR: Section find for course prefix %s failed! Response code was: %s", coursePrefix, res.Status)
				// Wait longer if 3 retries fail; we've probably been IP ratelimited...
				if numRetries >= 3 {
					log.Printf("WARNING: More than 3 retries have failed. Waiting for 5 minutes before attempting further retries.")
					time.Sleep(5 * time.Minute)
				} else {
					log.Printf("Getting new token and retrying in 3 seconds...")
					time.Sleep(3 * time.Second)
				}
				coursebookHeaders = utils.RefreshToken(chromedpCtx)
				// Give coursebook some time to recognize the new token
				time.Sleep(500 * time.Millisecond)
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

		// Get HTML data for all section IDs
		sectionsInCoursePrefix := 0
		for sectionIndex, id := range sectionIDs {

			// Get section info
			// Worth noting that the "req" and "div" params in the request below don't actually seem to matter... consider them filler to make sure the request goes through
			queryStr := fmt.Sprintf("id=%s&req=0bd73666091d3d1da057c5eeb6ef20a7df3CTp0iTMYFuu9paDeUptMzLYUiW4BIk9i8LIFcBahX2E2b18WWXkUUJ1Y7Xq6j3WZAKPbREfGX7lZY96lI7btfpVS95YAprdJHX9dc5wM=&action=section&div=r-62childcontent", id)

			// Try HTTP request, retrying if necessary
			res, err := utils.RetryHTTP(func() *http.Request {
				req, err := http.NewRequest("POST", "https://coursebook.utdallas.edu/clips/clip-cb11-hat.zog", strings.NewReader(queryStr))
				if err != nil {
					panic(err)
				}
				req.Header = coursebookHeaders
				return req
			}, cli, func(res *http.Response, numRetries int) {
				log.Printf("ERROR: Section id lookup for id %s failed! Response code was: %s", id, res.Status)
				// Wait longer if 3 retries fail; we've probably been IP ratelimited...
				if numRetries >= 3 {
					log.Printf("WARNING: More than 3 retries have failed. Waiting for 5 minutes before attempting further retries.")
					time.Sleep(5 * time.Minute)
				} else {
					log.Printf("Getting new token and retrying in 3 seconds...")
					time.Sleep(3 * time.Second)
				}
				coursebookHeaders = utils.RefreshToken(chromedpCtx)
				// Give coursebook some time to recognize the new token
				time.Sleep(500 * time.Millisecond)
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
		log.Printf("\nFinished scraping course prefix %s. Got %d sections.", coursePrefix, sectionsInCoursePrefix)
		totalSections += sectionsInCoursePrefix
	}
	log.Printf("\nDone scraping term! Scraped a total of %d sections.", totalSections)

}
