/*
	This file contains the code for the academic calendars scaper.
*/

package scrapers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

type AcademicCalendar struct {
	Title string
	Href  string
	Time  string
}

func ScrapeAcademicCalendars(outDir string) {
	// Start chromedp
	chromedpCtx, cancel := utils.InitChromeDp()

	// Get sub folder from output folder
	outSubDir := filepath.Join(outDir, "academicCalendars")

	// Make output folder
	err := os.MkdirAll(outSubDir, 0777)
	if err != nil {
		panic(err)
	}

	// Go to listings page
	chromedp.RunResponse(chromedpCtx,
		chromedp.Navigate(`https://www.utdallas.edu/academics/calendar/`),
	)

	// Extract data from links
	// Current
	academicCalendars := []AcademicCalendar{AcademicCalendar{"", "", "current"}}
	chromedp.Run(chromedpCtx, chromedp.TextContent("h2.wp-block-heading", &academicCalendars[0].Title, chromedp.ByQuery))
	var currentNode []*cdp.Node
	chromedp.Run(chromedpCtx, chromedp.Nodes("a.wp-block-button__link", &currentNode, chromedp.ByQuery))
	for i := 0; i < len(currentNode[0].Attributes); i += 2 {
		if currentNode[0].Attributes[i] == "href" {
			academicCalendars[0].Href = currentNode[0].Attributes[i+1]
		}
	}

	// Future list
	var futureNodes []*cdp.Node
	chromedp.Run(chromedpCtx,
		chromedp.Nodes(`//h2[normalize-space(text())="Future Terms"]/following-sibling::ul[1]//a`, &futureNodes, chromedp.BySearch),
	)
	academicCalendars = append(academicCalendars, extractTextAndHref(futureNodes, "future", chromedpCtx)...)

	// Past list
	var pastNodes []*cdp.Node
	chromedp.Run(chromedpCtx,
		chromedp.Nodes(`//h2[normalize-space(text())="Past Terms"]/following-sibling::div[1]//a`, &pastNodes, chromedp.BySearch),
	)
	academicCalendars = append(academicCalendars, extractTextAndHref(pastNodes, "past", chromedpCtx)...)

	// Don't need ChromeDP anymore
	cancel()

	// Download all PDFs
	for _, academicCalendar := range academicCalendars {
		downloadPdfFromBox(academicCalendar.Href, academicCalendar.Time+"-"+academicCalendar.Title, outSubDir)
	}
}

func extractTextAndHref(nodes []*cdp.Node, time string, chromedpCtx context.Context) []AcademicCalendar {
	output := []AcademicCalendar{}

	// Extract href and text
	for _, n := range nodes {
		var href, text string
		// Get href attribute
		for i := 0; i < len(n.Attributes); i += 2 {
			if n.Attributes[i] == "href" {
				href = n.Attributes[i+1]
			}
		}
		// Get inner text
		chromedp.Run(chromedpCtx, chromedp.TextContent(fmt.Sprintf(`a[href="%s"]`, href), &text, chromedp.ByQuery))

		output = append(output, AcademicCalendar{text, href, time})
	}

	return output
}

func downloadPdfFromBox(href string, filename string, outDir string) {
	// Create blank file
	out, err := os.Create(filepath.Join(outDir, fmt.Sprintf("%s.pdf", filename)))
	if err != nil {
		panic(err)
	}
	defer out.Close()

	// Pull ID from link
	parsedLink, err := url.Parse(href)
	if err != nil {
		panic(err)
	}
	fileId := path.Base(parsedLink.Path)

	// Use box download link with ID
	resp, err := http.Get(fmt.Sprintf("https://utdallas.box.com/shared/static/%s.pdf", fileId))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Output response to blank file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		panic(err)
	}

	log.Printf("Scraped academic calendar %s!", filename)
}
