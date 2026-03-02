package scrapers

import (
	"fmt"
	"log"

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

	// Write raw HTML to file
	outPath := fmt.Sprintf("%s/degreesScraped.html", outDir)
	utils.WriteJSON(outPath, html)

	log.Printf("Finished scraping discount page successfully!\n\n")
}
