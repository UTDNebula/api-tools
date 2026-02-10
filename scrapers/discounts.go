/*
	This file contains the code for the discount programs scraper.
*/

package scrapers

import (
	"context"
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

	// Create a custom chromedp context with suppressed error logging
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", utils.Headless),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("log-level", "3"),             // Suppress most logs (0=verbose, 3=fatal only)
		chromedp.Flag("disable-web-security", true), // Bypass CORS and security restrictions
		chromedp.Flag("disable-features", "IsolateOrigins,site-per-process,PrivateNetworkAccessPermissionPrompt"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	// Create context with discarded logger
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(string, ...interface{}) {}))
	defer cancel()

	log.Println("Loading discount page...")
	// Navigate to the discount page
	if err := chromedp.Run(ctx,
		chromedp.Navigate(discountUrl),
		chromedp.WaitReady("body"),
	); err != nil {
		panic(err)
	}

	// Wait for the content to load
	time.Sleep(2 * time.Second)

	// Get the HTML content
	var html string
	if err := chromedp.Run(ctx, chromedp.InnerHTML("body", &html)); err != nil {
		panic(err)
	}

	// Write raw HTML to file
	outPath := fmt.Sprintf("%s/discountsScraped.html", outDir)
	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		panic(err)
	}

	log.Printf("Finished scraping discount page successfully!\n\n")
}
