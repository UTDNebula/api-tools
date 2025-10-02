package parser

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/nebula-api/api/schema"
)

var (
	// Sections dictionary for mapping UUIDs to a *schema.Section
	Sections = make(map[primitive.ObjectID]*schema.Section)

	// Courses dictionary for keys (Internal_course_number +  Catalog_year) to a *schema.Course
	Courses = make(map[string]*schema.Course)

	// Professors dictionary for keys (First_name +   Last_name) to a *schema.Professor
	Professors = make(map[string]*schema.Professor)

	//CourseIDMap auxiliary dictionary for mapping UUIDs to a *schema.Course
	CourseIDMap = make(map[primitive.ObjectID]string)

	//ProfessorIDMap auxiliary dictionary for mapping UUIDs to a *schema.Professor
	ProfessorIDMap = make(map[primitive.ObjectID]string)

	// ReqParsers dictionary mapping course UUIDs to the func() that parsers its Reqs
	ReqParsers = make(map[primitive.ObjectID]func())

	// GradeMap mappings for section grade distributions, mapping is MAP[SEMESTER] -> MAP[SUBJECT + NUMBER + SECTION] -> GRADE DISTRIBUTION
	GradeMap map[string]map[string][]int

	// timeLocation Time location for dates (uses America/Chicago tz database zone for CDT which accounts for daylight saving)
	timeLocation, timeError = time.LoadLocation("America/Chicago")
)

// Parse Externally exposed parse function
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
	loadProfiles(filepath.Join(inDir, "profiles"))

	// Find paths of all scraped data
	paths := utils.GetAllFilesWithExtension(filepath.Join(inDir, "coursebook"), ".html")
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
