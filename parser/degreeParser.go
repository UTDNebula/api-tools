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
	Title           string        `bson:"name" json:"name"`
	School          string        `bson:"school" json:"school"`
	DegreeLevels    []DegreeLevel `bson:"degree_levels" json:"degree_levels"`
	AreasOfInterest []string      `bson:"areas_of_interest" json:"areas_of_interest"`
}

type DegreeLevel struct {
	Level          string `bson:"level" json:"level"`
	PublicUrl      string `bson:"public_url" json:"public_url"`
	CipCode        string `bson:"cip_code" json:"cip_code"`
	StemDesignated bool   `bson:"stem_designated" json:"stem_designated"`
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

	var degreeLevels []DegreeLevel
	content.Find("div .element-item.all.alldegrees.allschools.academic.bass.masters").Each(func(i int, s *goquery.Selection) {
		header := s.Find("div > h3").Parent()
		title := header.Find("h3")
		school := header.Find("div.school")

		s.Find("div.degrees > a.footnote").Each(func(j int, degreeLink *goquery.Selection) {
			level, exists := degreeLink.Attr("alt")
			if !exists {
				log.Println("error parsing alt value:")
			}

			urlForDegree, exists := degreeLink.Attr("href")
			if !exists {
				log.Println("error parsing href value:")
			}

			cipCode := degreeLink.Find("div.cip_code")
			stemDesignated := degreeLink.Find("div.footnote").Last() // There is only 1 element named STEM-Designated

			degreeLevels = append(degreeLevels, DegreeLevel{
				Level:          level,
				PublicUrl:      strings.TrimSpace(urlForDegree),
				CipCode:        strings.TrimSpace(cipCode.Text()),
				StemDesignated: isNotBlank(strings.TrimSpace(stemDesignated.Text())),
			})
		})

		areasOfInterest := s.Find("div.areas_of_interest.d-none").First()

		d := Degree{
			Title:           strings.TrimSpace(title.Text()),
			School:          strings.TrimSpace(school.Text()),
			DegreeLevels:    degreeLevels,
			AreasOfInterest: parseAreasOfInterest(areasOfInterest.Text()),
		}

		marshalled, err := json.MarshalIndent(d, "", "\t")
		if err != nil {
			panic("could not convert degree to JSON format")
		}

		/* Debug */
		log.Print(string(marshalled))

		/* Write to output File */
		outFile, err := os.Create(fmt.Sprintf("%s/degrees.json", outDir))
		if err != nil {
			log.Fatalf("could not create output file: %s", err)
		}

		_, err = outFile.Write(marshalled)
		if err != nil {
			log.Fatalf("could not write to output file: %s", err)
		}
	})
}

func isNotBlank(s string) bool {
	return s != "" && len(strings.TrimSpace(s)) > 0
}

func parseAreasOfInterest(tags string) []string {
	return strings.Split(strings.TrimSpace(tags), ",")
}

// Generate all possible combinations of filters
/*
func GenerateAllCombinations() []map[string]string {
	schools := []string{"bass", "jindal", ""}
	levels := []string{"bachelors", "masters", ""}
	depts := []string{"academic", ""}

	var combinations []map[string]string
}
*/
