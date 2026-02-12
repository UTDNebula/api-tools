package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Degree struct {
	Id           string        `bson:"id" json:"id"`
	Title        string        `bson:"name" json:"name"`
	School       string        `bson:"school" json:"school"`
	Department   string        `bson:"department" json:"department"`
	StemDesigned bool          `bson:"stem_designated" json:"stem_designated"`
	DegreeLevels []DegreeLevel `bson:"degreeLevels" json:"degreeLevels"`
	PublicUrl    string        `bson:"public_url" json:"public_url"`
}

type DegreeLevel struct {
	Level        string `bson:"level" json:"level"`
	Abbreviation string `bson:"abbreviation" json:"abbreviation"`
}

func ParseDegrees(inDir string, outDir string) {
	// Read the scraped HTML file
	htmlPath := fmt.Sprintf("%s/degreesScraped.html", inDir)
	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		panic(err)
	}

	log.Println("Parsing Degrees...")

	page, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlBytes)))

	// Find main content
	content := page.Find("article .col-sm-12").First()
	if content.Length() == 0 {
		panic("failed to find content area")
	}

	//var degrees []Degrees

	content.Find("div .element-item.all.alldegrees.allschools.academic.bass.masters").
		Each(func(i int, s *goquery.Selection) {
			degree := Degree{}

			header := s.Find("div.degreeTitle, div")
			title := header.Find("h3")
			school := header.Find("div.school")
			//schoolLink := header.Find("div.school, a")

			degree.Title = strings.TrimSpace(title.Text())
			degree.School = strings.TrimSpace(school.Text())

			marshalled, err := json.MarshalIndent(degree, "", "\t")
			if err != nil {
				panic("could not convert degree to JSON format")
			}

			log.Print(string(marshalled))
		})
}
