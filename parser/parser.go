package parser

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/UTDNebula/api-tools/utils"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/nebula-api/api/schema"
)

// Main dictionaries for mapping unique keys to the actual data
var Sections = make(map[schema.IdWrapper]*schema.Section)
var Courses = make(map[string]*schema.Course)
var Professors = make(map[string]*schema.Professor)

// Auxilliary dictionaries for mapping the generated ObjectIDs to the keys used in the above maps, used for validation purposes
var CourseIDMap = make(map[schema.IdWrapper]string)
var ProfessorIDMap = make(map[schema.IdWrapper]string)

// Requisite parser closures associated with courses
var ReqParsers = make(map[schema.IdWrapper]func())

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
		log.Printf("Loaded grade distributions for %d semesters.\n\n", len(GradeMap))
	}

	// Try to load any existing profile data
	loadProfiles(inDir)

	// Find paths of all scraped data
	paths := utils.GetAllFilesWithExtension(inDir, ".html")
	if !skipValidation {
		log.Printf("Parsing and validating %d files...\n", len(paths))
	} else {
		log.Printf("Parsing %d files WITHOUT VALIDATION...\n", len(paths))
	}

	// Parse all data
	for _, path := range paths {
		parse(path)
	}

	log.Printf("\nParsing complete. Created %d courses, %d sections, and %d professors.\n", len(Courses), len(Sections), len(Professors))

	log.Print("\nParsing course requisites...\n")

	// Initialize matchers at runtime for requisite parsing; this is necessary to avoid circular reference errors with compile-time initialization
	initMatchers()

	for _, course := range Courses {
		ReqParsers[course.Id]()
	}
	log.Print("Finished parsing course requisites!\n")

	if !skipValidation {
		log.Print("\nStarting validation stage...\n")
		validate()
		log.Print("\nValidation complete!\n")
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
	log.Printf("Parsing %s...\n", path)

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

	// Get the rows of the info table
	infoTable := doc.FindMatcher(goquery.Single("table.courseinfo__overviewtable > tbody"))
	infoRows := infoTable.ChildrenFiltered("tr")

	var syllabusURI string

	// Dictionary to hold the row data, keyed by row header
	rowInfo := make(map[string]string, len(infoRows.Nodes))

	// Populate rowInfo
	infoRows.Each(func(_ int, row *goquery.Selection) {
		rowHeader := utils.TrimWhitespace(row.FindMatcher(goquery.Single("th")).Text())
		rowData := row.FindMatcher(goquery.Single("td"))
		rowInfo[rowHeader] = utils.TrimWhitespace(rowData.Text())
		// Get syllabusURI from syllabus row link
		if rowHeader == "Syllabus:" {
			syllabusURI, _ = rowData.FindMatcher(goquery.Single("a")).Attr("href")
		}
	})

	// Get the rows of the class info subtable
	infoSubTable := infoTable.FindMatcher(goquery.Single("table.courseinfo__classsubtable > tbody"))
	infoRows = infoSubTable.ChildrenFiltered("tr")

	// Dictionary to hold the class info, keyed by data label
	classInfo := make(map[string]string)

	// Populate classInfo
	infoRows.Each(func(_ int, row *goquery.Selection) {
		rowHeaders := row.Find("td.courseinfo__classsubtable__th")
		rowHeaders.Each(func(_ int, header *goquery.Selection) {
			headerText := utils.TrimWhitespace(header.Text())
			dataText := utils.TrimWhitespace(header.Next().Text())
			classInfo[headerText] = dataText
		})
	})

	// Get the class and course num by splitting classInfo value
	classAndCourseNum := strings.Split(classInfo["Class/Course Number:"], " / ")
	classNum := classAndCourseNum[0]
	courseNum := utils.TrimWhitespace(classAndCourseNum[1])

	// Figure out the academic session associated with this specific course/Section
	session := getAcademicSession(rowInfo, classInfo)

	// Try to create the course and section based on collected info
	courseRef := parseCourse(courseNum, session, rowInfo, classInfo)
	parseSection(courseRef, classNum, syllabusURI, session, rowInfo, classInfo)
	log.Print("Parsed!\n")
}
