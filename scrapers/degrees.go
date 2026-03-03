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
	const URL = "https://academics.utdallas.edu/degrees/"

	ctx, cancel := utils.InitChromeDp()
	defer cancel()

	var html string
	log.Println("Scraping Degrees!")
	err := chromedp.Run(ctx,
		chromedp.Navigate(URL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.InnerHTML("article .col-sm-12", &html),
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
