package scrapers

import (
	"fmt"
	"log"
	"os"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/chromedp/chromedp"
)

func ScrapeDegrees(outDir string) {
	// Define the URL
	const DEGREE_URL = "https://academics.utdallas.edu/degrees/"

	chromedpCtx, cancel := utils.InitChromeDp()
	defer cancel()

	var html string
	log.Println("Scraping Degrees!")
	err := chromedp.Run(chromedpCtx,
		chromedp.Navigate(DEGREE_URL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.OuterHTML("article .col-sm-12", &html),
	)
	if err != nil {
		log.Panicf("failed to scrape: %v", err)
	}

	// Write raw HTML to file
	outPath := fmt.Sprintf("%s/degreesScraped.html", outDir)
	err = os.WriteFile(outPath, []byte(html), 0644)
	if err != nil {
		panic(err)
	}

	log.Printf("Finished scraping discount page successfully!\n\n")
}
