package scrapers

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/UTDNebula/api-tools/parser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
)

func ScrapeEvals(inDir string) {

	// Load env vars
	if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
	}

	// Make sure chromedp is initialized
	chromedpCtx, cancel := initChromeDp()
	defer cancel()

	// Get a token from coursebook, because we need that for the ues-report endpoint to work properly
	refreshToken(chromedpCtx)

	// Get all section filepaths for section ids
	sectionPaths := parser.GetAllSectionFilepaths(inDir)
	for i, path := range sectionPaths {

		_, fileName := filepath.Split(path)
		sectionID := fileName[:len(fileName)-5]

		log.Printf("Finding eval for %s\n", sectionID)

		// Get eval info
		evalURL := fmt.Sprintf("https://coursebook.utdallas.edu/ues-report/%s", sectionID)
		// Navigate to eval URL and pull all HTML
		var html string
		_, err := chromedp.RunResponse(chromedpCtx, chromedp.Tasks{
			chromedp.Navigate(evalURL),
			chromedp.QueryAfter("table", func(ctx context.Context, eci runtime.ExecutionContextID, n ...*cdp.Node) error {
				if len(n) > 0 {
					// Create and write eval HTML to file
					chromedp.OuterHTML("html", &html).Do(ctx)
					fptr, err := os.Create(strings.Replace(path, ".html", ".html.eval", 1))
					if err != nil {
						panic(err)
					}
					if _, err := fptr.WriteString(html); err != nil {
						panic(err)
					}
					fptr.Close()
					log.Print("Eval found and downloaded!\n")
					return err
				} else {
					log.Print("No eval found!\n")
					return nil
				}
			}, chromedp.AtLeast(0)),
		})
		if err != nil {
			panic(err)
		}

		// Avoid the ratelimit by refreshing the token periodically
		if i%30 == 0 && i != 0 {
			refreshToken(chromedpCtx)
			// Give coursebook some time to recognize the new token
			time.Sleep(1250 * time.Millisecond)
		}

		time.Sleep(100 * time.Millisecond)
	}
}
