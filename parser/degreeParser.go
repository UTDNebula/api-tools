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
	// Read the scraped HTML file
	htmlPath := fmt.Sprintf("%s/degreesScraped.html", inDir)
	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		log.Fatalf("Could not read HTML file: %v", err)
	}
	utils.VPrintf("Read %d bytes from %s", len(htmlBytes), htmlPath)

	// Parse the document
	page, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlBytes)))
	if err != nil {
		log.Fatalf("Failed to parse HTML: %v", err)
	}

	// Find main content
	content := page.Find(".col-sm-12").First()
	if content.Length() == 0 {
		log.Fatalf("failed to find content area")
	}
	utils.VPrintf("Found main content area")

	// Generate all possible combinations of degree filters
	// This is done to cover all degrees from different schools e.g. ECS, NSM, etc
	allProgramHTMLs := generateAllCombinations()
	utils.VPrintf("Generated %d program combinations to search", len(allProgramHTMLs))

	var allPrograms []schema.AcademicProgram
	for _, programHTML := range allProgramHTMLs {
		content.Find(programHTML).Each(func(i int, s *goquery.Selection) {
			extractProgram(s, &allPrograms)
		})
	}
	utils.VPrintf("Extracted %d programs", len(allPrograms))

	// Write to output file
	utils.WriteJSON(filepath.Join(outDir, "degrees.json"), allPrograms)

	utils.VPrintf("Successfully wrote degrees to %s/degrees.json", outDir)
}

func extractProgram(selection *goquery.Selection, programs *[]schema.AcademicProgram) {
	header := selection.Find("div > h3").Parent()
	title := header.Find("h3")
	school := header.Find("div.school")
	utils.VPrintf("Extracting program: %s (%s)", strings.TrimSpace(title.Text()), strings.TrimSpace(school.Text()))

	var degrees []schema.Degree
	selection.Find("div.degrees > a.footnote").Each(func(j int, degreeLink *goquery.Selection) {
		// The alt attribute represents the Degree Level
		// Example: BS in Buisness Administration
		degreeLevel, exists := degreeLink.Attr("alt")
		if !exists {
			log.Println("error parsing alt value:")
			return
		}
		// Normalize Degree Level to just represent Level. Ex: BS, BA, PhD, etc
		degreeLevel = strings.TrimSpace(degreeLevel[:3])

		// Extracts the URL to the degree's page.
		urlForDegree, exists := degreeLink.Attr("href")
		if !exists {
			log.Println("Error parsing href value:")
			return
		}

		// Extracts Classification of Instructional Programs Codes.
		// These codes provide a standardized system for reporting data about
		// academic programs across different colleges and universities.
		cipCode := degreeLink.Find("div.cip_code")

		// Extracts the footnote from the degree HTML
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
	// Example for Computer Science: Artificial intelligence, AI, computer science, software, robotics,
	areasOfInterest := selection.Find("div.areas_of_interest.d-none").First()

	newProgram := schema.AcademicProgram{
		Title:         strings.TrimSpace(title.Text()),
		School:        strings.TrimSpace(school.Text()),
		DegreeOptions: degrees,
		// Normalize to lowercase and split comma-separated values
		AreasOfInterest: strings.Split(strings.TrimSpace(strings.ToLower(areasOfInterest.Text())), ", "),
	}
	utils.VPrintf("  Areas of interest: %d topics", len(newProgram.AreasOfInterest))

	*programs = append(*programs, newProgram)
}

// Generates a list of all possible HTML endpoints for a degree from the HTML Page
//
// Each endpoint corresponds to a specific school,
// combining it with common CSS selectors used in the document structure
func generateAllCombinations() []string {
	// List of schools for which we need to generate combination selectors
	schools := []string{"bass", "jindal", "nsm", "ecs", "bbs", "epps"}

	var combinations []string

	// Loop through each school and generate the corresponding HTML selector
	baseEndpoint := "div .element-item.all.alldegrees.allschools.academic."
	for _, s := range schools {
		combinations = append(combinations, fmt.Sprintf("%s%s", baseEndpoint, s))
	}

	return combinations
}
