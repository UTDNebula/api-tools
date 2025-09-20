/*
	This file contains the code for the academic calendars scaper.
*/

package scrapers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

func ScrapeAcademicCalendars(outDir string) {
	// Get sub folder from output folder
	outSubDir := filepath.Join(outDir, "academicCalendars")

	// Make output folder
	err := os.MkdirAll(outSubDir, 0777)
	if err != nil {
		panic(err)
	}

	downloadPdfFromBox("https://utdallas.app.box.com/s/30v4pb6o5xjzncl5nfrgcw5wcrjz79kn", "hi", outSubDir)
}

func downloadPdfFromBox(link string, filename string, outDir string) {
	// Create blank file
	out, err := os.Create(filepath.Join(outDir, fmt.Sprintf("%s.pdf", filename)))
	if err != nil {
		panic(err)
	}
	defer out.Close()

	// Pull ID from link
	parsedLink, err := url.Parse(link)
	if err != nil {
		panic(err)
	}
	fileId := path.Base(parsedLink.Path)

	// Use box download link with ID
	resp, err := http.Get(fmt.Sprintf("https://utdallas.box.com/shared/static/%s.pdf", fileId))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Output response to blank file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		panic(err)
	}

	log.Printf("Scraped academic calendar %s!", filename)
}
