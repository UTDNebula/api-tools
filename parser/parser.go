// Package parser converts scraped course and scheduling inputs into structured Nebula API schema documents.
package parser

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/nebula-api/api/schema"
)

var (
	// Sections maps section IDs to the associated section records.
	Sections = make(map[primitive.ObjectID]*schema.Section)

	// Courses maps catalog identifiers to course definitions.
	Courses = make(map[string]*schema.Course)

	// Professors maps professor names to professor documents.
	Professors = make(map[string]*schema.Professor)

	// CourseIDMap maps course IDs to their catalog keys.
	CourseIDMap = make(map[primitive.ObjectID]string)

	// ProfessorIDMap maps professor IDs to their lookup keys.
	ProfessorIDMap = make(map[primitive.ObjectID]string)

	// ReqParsers maps course IDs to requisite parser functions.
	ReqParsers = make(map[primitive.ObjectID]func())

	// GradeMap stores grade distributions keyed by semester and section identifier.
	GradeMap map[string]map[string][]int

	// timeLocation captures the America/Chicago location for timestamp normalization.
	timeLocation, timeError = time.LoadLocation("America/Chicago")
)

// Parse loads scraped course artifacts, applies parsing and validation, and persists structured results.
func Parse(inDir string, outDir string, csvPath string, skipValidation bool) {

	// Panic if timeLocation didn't load properly
	if timeError != nil {
		panic(timeError)
	}

	// Load grade data from csv in advance
	GradeMap = loadGrades(csvPath)
	if len(GradeMap) != 0 {
		log.Printf("Loaded grade distributions for %d semesters.", len(GradeMap))
	}

	// Try to load any existing profile data
	loadProfiles(inDir)

	// Find paths of all scraped data
	paths := utils.GetAllFilesWithExtension(inDir, ".html")
	if !skipValidation {
		log.Printf("Parsing and validating %d files...", len(paths))
	} else {
		log.Printf("Parsing %d files WITHOUT VALIDATION...", len(paths))
	}

	// Parse all data
	for _, path := range paths {
		parse(path)
	}

	log.Printf("\nParsing complete. Created %d courses, %d sections, and %d professors.", len(Courses), len(Sections), len(Professors))

	log.Print("\nParsing course requisites...")

	// Initialize matchers at runtime for requisite parsing; this is necessary to avoid circular reference errors with compile-time initialization
	initMatchers()

	for _, course := range Courses {
		ReqParsers[course.Id]()
	}
	log.Print("Finished parsing course requisites!")

	if !skipValidation {
		log.Print("\nStarting validation stage...")
		validate()
		log.Print("\nValidation complete!")
	}

	// Make outDir if it doesn't already exist
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	// Write validated data to output files
	utils.WriteJSON(fmt.Sprintf("%s/courses.json", outDir), utils.GetMapValues(Courses))
	utils.WriteJSON(fmt.Sprintf("%s/sections.json", outDir), utils.GetMapValues(Sections))
	utils.WriteJSON(fmt.Sprintf("%s/professors.json", outDir), utils.GetMapValues(Professors))
}

// parse is an internal helper function that parses a single HTML file.
// It opens the file, creates a goquery document, and calls parseSection to
// extract section data.
func parse(path string) {

	utils.VPrintf("Parsing %s...", path)

	// Open data file for reading
	fptr, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fptr.Close()

	// Create a goquery document for HTML parsing
	doc, err := goquery.NewDocumentFromReader(fptr)
	if err != nil {
		panic(err)
	}

	parseSection(getRowInfo(doc), getClassInfo(doc))

	utils.VPrint("Parsed!")
}

func getRowInfo(doc *goquery.Document) map[string]*goquery.Selection {
	infoRows := doc.FindMatcher(goquery.Single("table.courseinfo__overviewtable > tbody")).ChildrenFiltered("tr")
	rowInfo := make(map[string]*goquery.Selection, len(infoRows.Nodes))

	infoRows.Each(func(_ int, row *goquery.Selection) {
		rowHeader := utils.TrimWhitespace(row.FindMatcher(goquery.Single("th")).Text())
		rowInfo[rowHeader] = row.FindMatcher(goquery.Single("td"))

	})
	return rowInfo
}

func getClassInfo(doc *goquery.Document) map[string]string {
	infoRows := doc.FindMatcher(goquery.Single("table.courseinfo__classsubtable > tbody")).ChildrenFiltered("tr")
	classInfo := make(map[string]string, len(infoRows.Nodes))

	infoRows.Each(func(_ int, row *goquery.Selection) {
		rowHeaders := row.Find("td.courseinfo__classsubtable__th")
		rowHeaders.Each(func(_ int, header *goquery.Selection) {
			headerText := utils.TrimWhitespace(header.Text())
			dataText := utils.TrimWhitespace(header.Next().Text())
			classInfo[headerText] = dataText
		})
	})
	return classInfo
}
