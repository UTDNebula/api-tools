package scrapers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/chromedp/chromedp"
)

func ScrapeDegrees(outDir string) {
	// Define the URL
	const URL = "https://academics.utdallas.edu/degrees/#filter=.alldegrees.bass"

	ctx, cancel := utils.InitChromeDp()
	defer cancel()

	var html string
	log.Println("Scraping Degrees!")
	err := chromedp.Run(ctx,
		chromedp.Navigate(URL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		log.Panicf("failed to scrape: %v", err)
	}

	// Ensure the output directory exists
	outputPath := filepath.Join(outDir, "degrees")
	err = os.MkdirAll(outputPath, os.ModePerm)
	if err != nil {
		log.Panicf("failed to create directory: %v", err)
	}

	// Write raw HTML to file
	outPath := fmt.Sprintf("%s/degreesScraped.html", outDir)
	err = os.WriteFile(outPath, []byte(html), 0644)
	if err != nil {
		panic(err)
	}

	log.Printf("Finished scraping discount page successfully!\n\n")
}
