/*
	This file contains the code for the budgets scaper.
*/

package scrapers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

type Budget struct {
	Title string
	Href  string
	Type  string
}

func ScrapeBudgets(outDir string) {
	// Start chromedp
	chromedpCtx, cancel := utils.InitChromeDp()

	// Get sub folder from output folder
	outSubDir := filepath.Join(outDir, "budgets")

	// Make output folder
	os.RemoveAll(outSubDir)
	err := os.MkdirAll(outSubDir, 0777)
	if err != nil {
		panic(err)
	}

	// Go to listings page
	_, err = chromedp.RunResponse(chromedpCtx,
		chromedp.Navigate(`https://finance.utdallas.edu/for-others/public-reports/`),
	)
	if err != nil {
		panic(err)
	}

	// Selector for the scraping the budget nodes
	financialReportSel := `//h2[normalize-space(text())="Annual Financial Statements"]/following-sibling::ul[1]//a`
	budgetReportSel := `//h2[normalize-space(text())="Annual Budget Reports"]/following-sibling::div[1]//details//ul/li[1]/a`

	budgets := []Budget{}

	// Extract data from links
	// Annual Financial Statements
	var financialReportNodes []*cdp.Node
	err = chromedp.Run(chromedpCtx,
		chromedp.Nodes(financialReportSel, &financialReportNodes, chromedp.BySearch),
	)
	if err != nil {
		panic(err)
	}
	links := utils.ExtractTextAndHref(financialReportNodes, chromedpCtx)
	for _, link := range links {
		budgets = append(budgets, Budget{
			Title: link.Text,
			Href:  link.Href,
			Type:  "financialReport",
		})
	}

	// Annual Financial Statements
	var budgetReportNodes []*cdp.Node
	err = chromedp.Run(chromedpCtx,
		chromedp.Nodes(budgetReportSel, &budgetReportNodes, chromedp.BySearch),
	)
	if err != nil {
		panic(err)
	}
	links = utils.ExtractTextAndHref(budgetReportNodes, chromedpCtx)
	for _, link := range links {
		budgets = append(budgets, Budget{
			Title: link.Text,
			Href:  link.Href,
			Type:  "budgetReport",
		})
	}

	// Don't need ChromeDP anymore
	cancel()

	// Download all PDFs
	for _, budget := range budgets {
		downloadPdf(
			budget.Href,
			budget.Type+"-"+budget.Title,
			outSubDir,
		)
	}
}

func downloadPdf(href string, filename string, outDir string) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", href, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/110.0")
	req.Header.Set("Referer", "https://finance.utdallas.edu/for-others/public-reports/")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		panic(fmt.Errorf("Failed to download \"%s\": status code %d", filename, resp.StatusCode))
	}

	// Create blank file
	out, err := os.Create(filepath.Join(outDir, fmt.Sprintf("%s.pdf", filename)))
	if err != nil {
		panic(err)
	}
	defer out.Close()

	// Output response to blank file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Printf("Error saving %s: %v", filename, err)
		return
	}

	log.Printf("Scraped budget %s!", filename)

	time.Sleep(1 * time.Second)
}
