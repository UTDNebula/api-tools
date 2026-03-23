/*
	This file contains the code for the discount programs scraper.
*/

package scrapers

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/chromedp/chromedp"
)

const discountUrl = "https://sg.utdallas.edu/discount/"

// ScrapeDiscounts retrieves the discount programs page HTML and saves it.
func ScrapeDiscounts(outDir string) {
	// Ensure output directory exists
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	chromedpCtx, cancel := utils.InitChromeDp()
	defer cancel()

	log.Println("Loading discount page...")
	// Navigate to the discount page
	if _, err := chromedp.RunResponse(chromedpCtx,
		chromedp.Navigate(discountUrl),
		chromedp.WaitReady("body", chromedp.ByQuery),
	); err != nil {
		panic(err)
	}

	// Wait for the content to load
	time.Sleep(2 * time.Second)

	// Get the HTML content
	var html string
	if err := chromedp.Run(chromedpCtx, chromedp.InnerHTML("body", &html)); err != nil {
		panic(err)
	}

	// Write raw HTML to file
	outPath := fmt.Sprintf("%s/discountsScraped.html", outDir)
	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		panic(err)
	}

	log.Printf("Scraped discounts successfully!\n")
}
