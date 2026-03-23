package scrapers

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/chromedp/chromedp"
)

const degreesUrl = "https://academics.utdallas.edu/degrees/"

func ScrapeDegrees(outDir string) {
	// Ensure output directory exists
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	chromedpCtx, cancel := utils.InitChromeDp()
	defer cancel()

	log.Println("Scraping Degrees!")
	_, err = chromedp.RunResponse(chromedpCtx,
		chromedp.Navigate(degreesUrl),
		chromedp.WaitVisible("article .col-sm-12", chromedp.ByQuery),
	)
	if err != nil {
		log.Panicf("failed to scrape: %v", err)
	}

	// Wait for the article content to load
	time.Sleep(2 * time.Second)

	var html string
	err = chromedp.Run(chromedpCtx, chromedp.OuterHTML("article .col-sm-12", &html))
	if err != nil {
		log.Panicf("failed to scrape: %v", err)
	}

	// Write raw HTML to file
	outPath := fmt.Sprintf("%s/degreesScraped.html", outDir)
	err = os.WriteFile(outPath, []byte(html), 0644)
	if err != nil {
		panic(err)
	}

	log.Printf("Scraped degrees successfully!\n")
}
