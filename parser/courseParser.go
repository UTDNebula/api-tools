package parser

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/PuerkitoBio/goquery"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	// coursePrefixRegexp matches the course prefix and number (e.g., "CS 1337").
	coursePrefixRegexp = utils.Regexpf(`^%s`, utils.R_SUBJ_COURSE_CAP)

	// contactRegexp matches the contact hours and offering frequency from the course description
	// (e.g. "(12-34) SUS")
	contactRegexp = regexp.MustCompile(`\(([0-9]+)-([0-9]+)\)\s+([SUFY]+)`)
)

// parseCourse returns a pointer to the course specified by the
// provided information. If the associated course is not found in
// Courses, it will run getCourse and add the result to Courses.
func parseCourse(internalCourseNumber string, session schema.AcademicSession, rowInfo map[string]*goquery.Selection, classInfo map[string]string) *schema.Course {
	// Courses are internally keyed by their identifying fields
	catalogYear := getCatalogYear(session)
	subjectPrefix, courseNumber := getPrefixAndNumber(classInfo)
	courseMapKey := subjectPrefix + courseNumber + catalogYear

	// Don't recreate the course if it already exists
	course, courseExists := Courses[courseMapKey]
	if courseExists {
		return course
	}

	course = getCourse(internalCourseNumber, session, rowInfo, classInfo)

	// Get closure for parsing course requisites
	enrollmentReqs, hasEnrollmentReqs := rowInfo["Enrollment Reqs:"]
	ReqParsers[course.Id] = getReqParser(course, hasEnrollmentReqs, enrollmentReqs)

	Courses[courseMapKey] = course
	CourseIDMap[course.Id] = courseMapKey
	return course
}

// getCourse extracts course details from the provided information and creates a schema.Course object.
// This function does not modify any global state.
// Returns a pointer to the newly created schema.Course object.
func getCourse(internalCourseNumber string, session schema.AcademicSession, rowInfo map[string]*goquery.Selection, classInfo map[string]string) *schema.Course {
	subjectPrefix, courseNumber := getPrefixAndNumber(classInfo)
	catalogYear := getCatalogYear(session)

	course := schema.Course{
		Id:                     primitive.NewObjectID(),
		Subject_prefix:         subjectPrefix,
		Course_number:          courseNumber,
		Title:                  utils.TrimWhitespace(rowInfo["Course Title:"].Text()),
		Description:            utils.TrimWhitespace(rowInfo["Description:"].Text()),
		School:                 utils.TrimWhitespace(rowInfo["College:"].Text()),
		Credit_hours:           classInfo["Semester Credit Hours:"],
		Class_level:            classInfo["Class Level:"],
		Activity_type:          classInfo["Activity Type:"],
		Grading:                classInfo["Grading:"],
		Internal_course_number: internalCourseNumber,
		Catalog_year:           catalogYear,
		Sections:               []schema.SectionKey{},
	}

	// Try to get lecture/lab contact hours and offering frequency from course description
	contactMatches := contactRegexp.FindStringSubmatch(course.Description)
	// Length of contactMatches should be 4 upon successful match
	if len(contactMatches) == 4 {
		course.Lecture_contact_hours = contactMatches[1]
		course.Laboratory_contact_hours = contactMatches[2]
		course.Offering_frequency = contactMatches[3]
	}

	return &course
}

// getCatalogYear determines the catalog year from the academic session information.
// It assumes the session name starts with a 2-digit year and a semester character ('F', 'S', 'U').
// Fall sessions are associated with the same catalog year.
// Spring and Summer sessions are associated with the previous catalog year.
// (e.g. 20F = 20, 20S = 19)
func getCatalogYear(session schema.AcademicSession) string {
	sessionYear, err := strconv.Atoi(session.Name[0:2])
	if err != nil {
		panic(err)
	}

	sessionSemester := session.Name[2]
	switch sessionSemester {
	case 'F':
		return strconv.Itoa(sessionYear)
	case 'S', 'U':
		return strconv.Itoa(sessionYear - 1)
	default:
		panic(fmt.Errorf("encountered invalid session semester '%c'", sessionSemester))
	}
}

// getPrefixAndNumber returns the 2nd and 3rd matched values from a coursePrefixRegexp on
// classInfo["Class Section:"]. It expects classInfo to contain "Class Section:".
// If there are no matches, it panics.
func getPrefixAndNumber(classInfo map[string]string) (string, string) {
	sectionID, ok := classInfo["Class Section:"]
	if !ok {
		panic("could not find 'Class Section:' in classInfo")
	}

	// Get subject prefix and course number by doing a regexp match on the section id
	matches := coursePrefixRegexp.FindStringSubmatch(sectionID)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}

	panic("failed to parse course prefix and number")
}
