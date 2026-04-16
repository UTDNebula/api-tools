/*
	This file contains the code for the academic calendars scaper.
*/

package scrapers

import (
	"context"
	"encoding/csv"
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
	os.RemoveAll(outSubDir)
	err := os.MkdirAll(outSubDir, 0777)
	if err != nil {
		panic(err)
	}

	// Go to listings page
	_, err = chromedp.RunResponse(chromedpCtx,
		chromedp.Navigate(`https://www.utdallas.edu/academics/calendar/`),
	)
	if err != nil {
		panic(err)
	}

	// Selector for the scraping the calendar nodes
	currentSel := `a.wp-block-button__link`
	futureSel := `//h2[normalize-space(text())="Future Terms"]/following-sibling::ul[1]//a`
	pastSel := `//h2[normalize-space(text())="Past Terms"]/following-sibling::div[position()<=2]//a`

	// Extract data from links
	// Current
	academicCalendars := []AcademicCalendar{{"", "", "current"}}
	err = chromedp.Run(chromedpCtx,
		chromedp.TextContent("h2.wp-block-heading", &academicCalendars[0].Title, chromedp.ByQuery),
	)
	if err != nil {
		panic(err)
	}
	var currentNode []*cdp.Node
	err = chromedp.Run(chromedpCtx,
		chromedp.Nodes(currentSel, &currentNode, chromedp.ByQuery),
	)
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(currentNode[0].Attributes); i += 2 {
		if currentNode[0].Attributes[i] == "href" {
			academicCalendars[0].Href = currentNode[0].Attributes[i+1]
		}
	}

	// Future list
	var futureNodes []*cdp.Node
	err = chromedp.Run(chromedpCtx,
		chromedp.Nodes(futureSel, &futureNodes, chromedp.BySearch),
	)
	if err != nil {
		panic(err)
	}
	newCalendars := extractTextAndHref(futureNodes, "future", chromedpCtx)
	academicCalendars = append(academicCalendars, newCalendars...)

	// Past list
	var pastNodes []*cdp.Node
	err = chromedp.Run(chromedpCtx,
		chromedp.Nodes(pastSel, &pastNodes, chromedp.BySearch),
	)
	if err != nil {
		panic(err)
	}
	newCalendars = extractTextAndHref(pastNodes, "past", chromedpCtx)
	academicCalendars = append(academicCalendars, newCalendars...)

	// Don't need ChromeDP anymore
	cancel()

	// Create urls.csv to record each calendar's direct Box download URL.
	// The parser reads this file to attach the URL to each parsed AcademicCalendar.
	csvFile, err := os.Create(filepath.Join(outSubDir, "urls.csv"))
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	for _, academicCalendar := range academicCalendars {
		// Key matches the PDF filename (without extension) used by the parser.
		key := academicCalendar.Time + "-" + academicCalendar.Title

		// downloadPdfFromBox returns the Box URL it constructs, so we can record it in the CSV.
		boxUrl := downloadPdfFromBox(academicCalendar.Href, key, outSubDir)
		if err := writer.Write([]string{key, boxUrl}); err != nil {
			panic(err)
		}
	}
}

func extractTextAndHref(nodes []*cdp.Node, time string, chromedpCtx context.Context) []AcademicCalendar {
	output := []AcademicCalendar{}
	var err error

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
		err = chromedp.Run(chromedpCtx,
			chromedp.TextContent(fmt.Sprintf(`a[href="%s"]`, href), &text, chromedp.ByQuery),
		)
		if err != nil {
			panic(err)
		}
		output = append(output, AcademicCalendar{text, href, time})
	}

	return output
}

// downloadPdfFromBox downloads a PDF from a Box share link, saves it to outDir,
// and returns the direct Box download URL for use in urls.csv.
func downloadPdfFromBox(href string, filename string, outDir string) string {
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
	boxUrl := fmt.Sprintf("https://utdallas.box.com/shared/static/%s.pdf", fileId)
	resp, err := http.Get(boxUrl)
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
	return boxUrl
}
