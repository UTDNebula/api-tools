package parser

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

// Parses scarped degree HTML and outputs the data in JSON
func ParseDegrees(inDir string, outDir string) {
	htmlPath := fmt.Sprintf("%s/degreesScraped.html", inDir)
	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		log.Fatalf("Could not read HTML file: %v", err)
	}
	utils.VPrintf("Read %d bytes from %s", len(htmlBytes), htmlPath)

	page, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlBytes)))
	if err != nil {
		log.Fatalf("Failed to parse HTML: %v", err)
	}

	content := page.Find(".col-sm-12").First()
	if content.Length() == 0 {
		log.Fatalf("failed to find content area")
	}
	utils.VPrintf("Found main content area")

	// Generate all possible combinations of degree filters
	allProgramHTMLs := generateAllCombinations()
	utils.VPrintf("Generated %d program combinations to search", len(allProgramHTMLs))

	var allPrograms []schema.AcademicProgram
	for _, programHTML := range allProgramHTMLs {
		content.Find(programHTML).Each(func(i int, s *goquery.Selection) {
			extractProgram(s, &allPrograms)
		})
	}

	// Write to output file
	err = utils.WriteJSON(filepath.Join(outDir, "degrees.json"), allPrograms)
	if err != nil {
		log.Fatal("Failed to upload json")
	}

	utils.VPrintf("Successfully parsed %d degrees to %s/degrees.json", len(allPrograms), outDir)
}

// extractProgram parses the list of program to each degree
func extractProgram(selection *goquery.Selection, programs *[]schema.AcademicProgram) {
	header := selection.Find("div > h3").Parent()
	title := header.Find("h3")
	school := header.Find("div.school")
	utils.VPrintf("Extracting program: %s (%s)", strings.TrimSpace(title.Text()), strings.TrimSpace(school.Text()))

	var degrees []schema.Degree

	selection.Find("div.degrees > a.footnote").Each(func(j int, degreeLink *goquery.Selection) {
		// Ex: BS, BA, PhD, etc
		degreeLevel, exists := degreeLink.Attr("alt")
		if !exists {
			log.Println("error parsing alt value:")
			return
		}
		degreeLevel = strings.TrimSpace(degreeLevel[:3])

		// Extracts the URL to the degree's page.
		urlForDegree, exists := degreeLink.Attr("href")
		if !exists {
			log.Println("Error parsing href value:")
			return
		}

		// Extracts Classification of Instructional Programs Codes.
		cipCode := degreeLink.Find("div.cip_code")

		// Relevant footnotes are 'STEM-Designated' and 'Joint Program'
		footnote := degreeLink.Find("div.footnote")

		degrees = append(degrees, schema.Degree{
			Level:          degreeLevel,
			PublicUrl:      strings.TrimSpace(urlForDegree),
			CipCode:        strings.TrimSpace(cipCode.Text()),
			StemDesignated: strings.Contains(strings.TrimSpace(footnote.Text()), "STEM-Designated"),
			JointProgram:   strings.Contains(strings.TrimSpace(footnote.Text()), "Joint Program"),
		})
	})
	utils.VPrintf("  Found %d degrees", len(degrees))

	// Extracts a list of tags that correlate to what might interest a student
	areasOfInterest := selection.Find("div.areas_of_interest.d-none").First()

	newProgram := schema.AcademicProgram{
		Title:           strings.TrimSpace(title.Text()),
		School:          strings.TrimSpace(school.Text()),
		DegreeOptions:   degrees,
		AreasOfInterest: parseAreasOfInterest(areasOfInterest.Text()),
	}
	utils.VPrintf("  Areas of interest: %d topics", len(newProgram.AreasOfInterest))

	*programs = append(*programs, newProgram)
}

// generateAllCombinations gets a list of all possible sel for a degree from the HTML
func generateAllCombinations() []string {
	schools := []string{"bass", "jindal", "nsm", "ecs", "bbs", "epps"}

	var combinations []string

	// Generate HTML selector for each schools
	baseSel := "div .element-item.all.alldegrees.allschools.academic."
	for _, s := range schools {
		combinations = append(combinations, fmt.Sprintf("%s%s", baseSel, s))
	}

	return combinations
}

// parseAreasOfInterest parses string to array
func parseAreasOfInterest(areasOfInterest string) []string {
	trimmed := strings.TrimSpace(strings.ToLower(areasOfInterest))
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, ", ")
}
