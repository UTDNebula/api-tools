/*
	This file contains the code for the academic calendars scaper.
*/

package scrapers

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

func ScrapeAcademicCalendars(outDir string) {
	// Start chromedp
	chromedpCtx, cancel := utils.InitChromeDp()
	defer cancel()

	// Make output folder
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fullDir := filepath.Join(wd, outDir)

	downloadPdfFromBox("https://utdallas.app.box.com/s/30v4pb6o5xjzncl5nfrgcw5wcrjz79kn", "hi", fullDir, chromedpCtx)
}

func downloadPdfFromBox(url string, filename string, fullDir string, ctx context.Context) {
	// Set up a channel, so we can block later while we monitor the download progress
	done := make(chan string, 1)
	// Set up a listener to watch the download events and close the channel when complete
	chromedp.ListenTarget(ctx, func(v interface{}) {
		if ev, ok := v.(*browser.EventDownloadProgress); ok {
			completed := "(unknown)"
			if ev.TotalBytes != 0 {
				completed = fmt.Sprintf("%0.2f%%", ev.ReceivedBytes/ev.TotalBytes*100.0)
			}
			log.Printf("state: %s, completed: %s\n", ev.State.String(), completed)
			if ev.State == browser.DownloadProgressStateCompleted {
				done <- ev.GUID
				close(done)
			}
		}
	})

	// Navigate and trigger download
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		browser.
			SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(fullDir).
			WithEventsEnabled(true),
		chromedp.WaitVisible(`button[aria-label="Download"]`, chromedp.ByQuery),
		chromedp.Click(`button[aria-label="Download"]`, chromedp.ByQuery),
	); err != nil {
		panic(err)
	}

	// Wait for the PDF
	guid := <-done

	// Rename
	oldPath := filepath.Join(fullDir, guid)
	newPath := filepath.Join(fullDir, filename) + ".pdf"
	err := os.Rename(oldPath, newPath)
	if err != nil {
		panic(err)
	}

	log.Printf("Scraped academic calendar %s!", filename)
}
