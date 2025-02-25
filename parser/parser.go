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

// Main dictionaries for mapping unique keys to the actual data
var Sections = make(map[primitive.ObjectID]*schema.Section)
var Courses = make(map[string]*schema.Course)
var Professors = make(map[string]*schema.Professor)

// Auxilliary dictionaries for mapping the generated ObjectIDs to the keys used in the above maps, used for validation purposes
var CourseIDMap = make(map[primitive.ObjectID]string)
var ProfessorIDMap = make(map[primitive.ObjectID]string)

// Requisite parser closures associated with courses
var ReqParsers = make(map[primitive.ObjectID]func())

// Grade mappings for section grade distributions, mapping is MAP[SEMESTER] -> MAP[SUBJECT + NUMBER + SECTION] -> GRADE DISTRIBUTION
var GradeMap map[string]map[string][]int

// Time location for dates (uses America/Chicago tz database zone for CDT which accounts for daylight saving)
var timeLocation, timeError = time.LoadLocation("America/Chicago")

// Externally exposed parse function
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

// Internal parse function
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

	// Dictionary to hold the row data, keyed by row header
	rowInfo := getRowInfo(doc)
	// Dictionary to hold the class info, keyed by data label
	classInfo := getClassInfo(doc)

	// Get the class and course num by splitting classInfo value

	parseSection(rowInfo, classInfo)
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
