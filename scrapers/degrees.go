package scrapers

import (
	"bufio"
	"log"
	"os"
	"path/filepath"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/chromedp/chromedp"
)

func ScrapeDegrees(outDir string) {
	// Define the URL (replace with actual URL)
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

	// Write HTML to file
	filename := filepath.Join(outputPath, "degrees.html")
	file, err := os.Create(filename)
	if err != nil {
		log.Panicf("failed to create file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write HTML content
	_, err = writer.WriteString(html)
	if err != nil {
		log.Panicf("failed to write HTML: %v", err)
	}

	log.Println("Successfully scraped and saved degrees data.")
}
