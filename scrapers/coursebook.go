/*
	This file contains the code for the coursebook scraper.
*/

package scrapers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
)

func initChromeDp() (chromedpCtx context.Context, cancelFnc context.CancelFunc) {
	log.Printf("Initializing chromedp...")
	headlessEnv, present := os.LookupEnv("HEADLESS_MODE")
	doHeadless, _ := strconv.ParseBool(headlessEnv)
	if present && doHeadless {
		chromedpCtx, cancelFnc = chromedp.NewContext(context.Background())
		log.Printf("Initialized chromedp!")
	} else {
		allocCtx, _ := chromedp.NewExecAllocator(context.Background())
		chromedpCtx, cancelFnc = chromedp.NewContext(allocCtx)
	}
	return
}

// This function generates a fresh auth token and returns the new headers
func refreshToken(chromedpCtx context.Context) map[string][]string {
	netID, present := os.LookupEnv("LOGIN_NETID")
	if !present {
		log.Panic("LOGIN_NETID is missing from .env!")
	}
	password, present := os.LookupEnv("LOGIN_PASSWORD")
	if !present {
		log.Panic("LOGIN_PASSWORD is missing from .env!")
	}

	utils.VPrintf("Getting new token...")
	_, err := chromedp.RunResponse(chromedpCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			err := network.ClearBrowserCookies().Do(ctx)
			return err
		}),
		chromedp.Navigate(`https://wat.utdallas.edu/login`),
		chromedp.WaitVisible(`form#login-form`),
		chromedp.SendKeys(`input#netid`, netID),
		chromedp.SendKeys(`input#password`, password),
		chromedp.WaitVisible(`input#login-button`),
		chromedp.Click(`input#login-button`),
		//chromedp.WaitVisible(`body`),
	)
	if err != nil {
		panic(err)
	}

	var cookieStrs []string
	_, err = chromedp.RunResponse(chromedpCtx,
		chromedp.Navigate(`https://coursebook.utdallas.edu/`),
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := network.GetCookies().Do(ctx)
			cookieStrs = make([]string, len(cookies))
			gotToken := false
			for i, cookie := range cookies {
				cookieStrs[i] = fmt.Sprintf("%s=%s", cookie.Name, cookie.Value)
				if cookie.Name == "PTGSESSID" {
					utils.VPrintf("Got new token: PTGSESSID = %s", cookie.Value)
					gotToken = true
				}
			}
			if !gotToken {
				return errors.New("failed to get a new token")
			}
			return err
		}),
	)
	if err != nil {
		panic(err)
	}

	return map[string][]string{
		"Host":            {"coursebook.utdallas.edu"},
		"User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/110.0"},
		"Accept":          {"text/html"},
		"Accept-Language": {"en-US"},
		"Content-Type":    {"application/x-www-form-urlencoded"},
		"Cookie":          cookieStrs,
		"Connection":      {"keep-alive"},
	}
}

func ScrapeCoursebook(term string, startPrefix string, outDir string) {

	// Load env vars
	if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
	}

	// Start chromedp
	chromedpCtx, cancel := initChromeDp()
	defer cancel()

	// Refresh the token
	refreshToken(chromedpCtx)

	log.Printf("Finding course prefix nodes...")

	var coursePrefixes []string
	var coursePrefixNodes []*cdp.Node

	// Get option elements for course prefix dropdown
	err := chromedp.Run(chromedpCtx,
		chromedp.Navigate("https://coursebook.utdallas.edu"),
		chromedp.Nodes("select#combobox_cp option", &coursePrefixNodes, chromedp.ByQueryAll),
	)

	if err != nil {
		log.Panic(err)
	}

	log.Println("Found the course prefix nodes!")

	log.Println("Finding course prefixes...")

	// Remove the first option due to it being empty
	coursePrefixNodes = coursePrefixNodes[1:]

	// Get the value of each option and append to coursePrefixes
	for _, node := range coursePrefixNodes {
		coursePrefixes = append(coursePrefixes, node.AttributeValue("value"))
	}

	log.Println("Found the course prefixes!")

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
		coursebookHeaders := refreshToken(chromedpCtx)
		// Give coursebook some time to recognize the new token
		time.Sleep(500 * time.Millisecond)
		// String builder to store accumulated course HTML data for both class levels
		courseBuilder := strings.Builder{}

		log.Printf("Finding sections for course prefix %s...", coursePrefix)

		// Get courses for term and prefix, split by grad and undergrad to avoid 300 section cap
		for _, clevel := range []string{"clevel_u", "clevel_g"} {
			queryStr := fmt.Sprintf("action=search&s%%5B%%5D=term_%s&s%%5B%%5D=%s&s%%5B%%5D=%s", term, coursePrefix, clevel)
			req, err := http.NewRequest("POST", "https://coursebook.utdallas.edu/clips/clip-cb11-hat.zog", strings.NewReader(queryStr))
			if err != nil {
				panic(err)
			}
			req.Header = coursebookHeaders
			res, err := cli.Do(req)
			if err != nil {
				panic(err)
			}
			if res.StatusCode != 200 {
				log.Panicf("ERROR: Section find failed! Status was: %s\nIf the status is 404, you've likely been IP ratelimited!", res.Status)
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
			req, err := http.NewRequest("POST", "https://coursebook.utdallas.edu/clips/clip-cb11-hat.zog", strings.NewReader(queryStr))
			if err != nil {
				panic(err)
			}
			req.Header = coursebookHeaders
			res, err := cli.Do(req)
			if err != nil {
				panic(err)
			}
			if res.StatusCode != 200 {
				log.Panicf("ERROR: Section id lookup for id %s failed! Status was: %s\nIf the status is 404, you've likely been IP ratelimited!", id, res.Status)
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
				coursebookHeaders = refreshToken(chromedpCtx)
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
