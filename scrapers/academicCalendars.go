/*
	This file contains the code for the academic calendars scaper.
*/

package scrapers

import (
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

func ScrapeAcademicCalendars(outDir string) error {
	// Start chromedp
	chromedpCtx, cancel := utils.InitChromeDp()
	defer cancel()

	// Get sub folder from output folder
	outSubDir := filepath.Join(outDir, "academicCalendars")

	// Make output folder
	os.RemoveAll(outSubDir)
	err := os.MkdirAll(outSubDir, 0777)
	if err != nil {
		return fmt.Errorf("create academic calendar output directory: %w", err)
	}

	// Go to listings page
	_, err = chromedp.RunResponse(chromedpCtx,
		chromedp.Navigate(`https://www.utdallas.edu/academics/calendar/`),
	)
	if err != nil {
		return fmt.Errorf("navigate to academic calendars: %w", err)
	}

	// Selector for the scraping the calendar nodes
	currentSel := `a.wp-block-button__link`
	futureSel := `//h2[normalize-space(text())="Future Terms"]/following-sibling::ul[1]//a`
	pastSel := `//h2[normalize-space(text())="Past Terms"]/following-sibling::div[1]//a`

	// Extract data from links
	// Current
	academicCalendars := []AcademicCalendar{{"", "", "current"}}
	err = chromedp.Run(chromedpCtx,
		chromedp.TextContent("h2.wp-block-heading", &academicCalendars[0].Title, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("read current calendar title: %w", err)
	}
	var currentNode []*cdp.Node
	err = chromedp.Run(chromedpCtx,
		chromedp.Nodes(currentSel, &currentNode, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("read current calendar link: %w", err)
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
		return fmt.Errorf("read future calendar links: %w", err)
	}
	links, err := utils.ExtractTextAndHref(futureNodes, chromedpCtx)
	if err != nil {
		return err
	}
	for _, link := range links {
		academicCalendars = append(academicCalendars, AcademicCalendar{
			Title: link.Text,
			Href:  link.Href,
			Time:  "future",
		})
	}

	// Past list
	var pastNodes []*cdp.Node
	err = chromedp.Run(chromedpCtx,
		chromedp.Nodes(pastSel, &pastNodes, chromedp.BySearch),
	)
	if err != nil {
		return fmt.Errorf("read past calendar links: %w", err)
	}
	links, err = utils.ExtractTextAndHref(pastNodes, chromedpCtx)
	if err != nil {
		return err
	}
	for _, link := range links {
		academicCalendars = append(academicCalendars, AcademicCalendar{
			Title: link.Text,
			Href:  link.Href,
			Time:  "past",
		})
	}

	// Don't need ChromeDP anymore
	cancel()

	// Download all PDFs
	for _, academicCalendar := range academicCalendars {
		if err := downloadPdfFromBox(
			academicCalendar.Href,
			academicCalendar.Time+"-"+academicCalendar.Title,
			outSubDir,
		); err != nil {
			return err
		}
	}
	return nil
}

func downloadPdfFromBox(href string, filename string, outDir string) error {
	// Create blank file
	out, err := os.Create(filepath.Join(outDir, fmt.Sprintf("%s.pdf", filename)))
	if err != nil {
		return fmt.Errorf("create calendar PDF: %w", err)
	}
	defer out.Close()

	// Pull ID from link
	parsedLink, err := url.Parse(href)
	if err != nil {
		return fmt.Errorf("parse calendar URL: %w", err)
	}
	fileId := path.Base(parsedLink.Path)

	// Use box download link with ID
	resp, err := http.Get(fmt.Sprintf("https://utdallas.box.com/shared/static/%s.pdf", fileId))
	if err != nil {
		return fmt.Errorf("download calendar PDF: %w", err)
	}
	defer resp.Body.Close()

	// Output response to blank file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("save calendar PDF: %w", err)
	}

	log.Printf("Scraped academic calendar %s!", filename)
	return nil
}
