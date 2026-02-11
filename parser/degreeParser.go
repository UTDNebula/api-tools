package parser

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func ParseDegrees(inDir string) {
	// Read the scraped HTML file
	htmlPath := fmt.Sprintf("%s/discountsScraped.html", inDir)
	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		panic(err)
	}

	log.Println("Parsing Degrees...")

	page, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlBytes)))

	// Find main content
	content := page.Find("col-sm-12").First()
	if content.Length() == 0 {
		panic("failed to find content area")
	}

	fmt.Print(content.Text())
}
