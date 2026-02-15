package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Program struct {
	Title           string   `bson:"name" json:"name"`
	School          string   `bson:"school" json:"school"`
	DegreeOptions   []Degree `bson:"degree_levels" json:"degree_levels"`
	AreasOfInterest []string `bson:"areas_of_interest" json:"areas_of_interest"`
}

type Degree struct {
	Level          string `bson:"level" json:"level"`
	PublicUrl      string `bson:"public_url" json:"public_url"`
	CipCode        string `bson:"cip_code" json:"cip_code"`
	StemDesignated bool   `bson:"stem_designated" json:"stem_designated"`
	JointProgram   bool   `bson:"joint_program" json:"joint_program"`
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
	if err != nil {
		panic(err)
	}

	// Find main content
	content := page.Find("article .col-sm-12").First()
	if content.Length() == 0 {
		panic("failed to find content area")
	}

	programsHTML := GenerateAllCombinations()

	var allPrograms []Program
	for _, programHTML := range programsHTML {
		content.Find(programHTML).Each(func(i int, s *goquery.Selection) {
			header := s.Find("div > h3").Parent()
			title := header.Find("h3")
			school := header.Find("div.school")
			var degrees []Degree
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
				footnote := degreeLink.Find("div.footnote") // There is either 1 element named STEM-Designated or no elements at all

				degrees = append(degrees, Degree{
					Level:          level,
					PublicUrl:      strings.TrimSpace(urlForDegree),
					CipCode:        strings.TrimSpace(cipCode.Text()),
					StemDesignated: strings.Contains(strings.TrimSpace(footnote.Text()), "STEM-Designated"),
					JointProgram:   strings.Contains(strings.TrimSpace(footnote.Text()), "Joint Program"),
				})
			})

			areasOfInterest := s.Find("div.areas_of_interest.d-none").First()

			newProgram := Program{
				Title:           strings.TrimSpace(title.Text()),
				School:          strings.TrimSpace(school.Text()),
				DegreeOptions:   degrees,
				AreasOfInterest: parseAreasOfInterest(areasOfInterest.Text()),
			}

			allPrograms = append(allPrograms, newProgram)
		})
	}

	marshalled, err := json.MarshalIndent(allPrograms, "", "\t")
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
	defer outFile.Close()

	_, err = outFile.Write(marshalled)
	if err != nil {
		log.Fatalf("could not write to output file: %s", err)
	}
}

func parseAreasOfInterest(tags string) []string {
	return strings.Split(strings.TrimSpace(tags), ", ")
}

// Generate all possible combinations of filters
func GenerateAllCombinations() []string {
	schools := []string{"bass", "jindal", "nsm", "ecs", "bbs", "epps"}

	var combinations []string

	for _, s := range schools {
		combinations = append(combinations, fmt.Sprintf("div .element-item.all.alldegrees.allschools.academic.%s", s))
	}

	return combinations
}
