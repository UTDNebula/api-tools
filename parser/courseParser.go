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

var coursePrefixRexp *regexp.Regexp = utils.Regexpf(`^%s`, utils.R_SUBJ_COURSE_CAP)
var contactRegexp *regexp.Regexp = regexp.MustCompile(`\(([0-9]+)-([0-9]+)\)\s+([SUFY]+)`)

func parseCourse(courseNum string, session schema.AcademicSession, rowInfo map[string]*goquery.Selection, classInfo map[string]string) *schema.Course {
	// Courses are internally keyed by their internal course number and the catalog year they're part of
	catalogYear := getCatalogYear(session)
	courseKey := courseNum + catalogYear

	// Don't recreate the course if it already exists
	course, courseExists := Courses[courseKey]
	if courseExists {
		return course
	}

	course = getCourse(courseNum, session, rowInfo, classInfo)

	// Get closure for parsing course requisites (god help me)
	enrollmentReqs, hasEnrollmentReqs := rowInfo["Enrollment Reqs:"]
	ReqParsers[course.Id] = getReqParser(course, hasEnrollmentReqs, enrollmentReqs)

	Courses[courseKey] = course
	CourseIDMap[course.Id] = courseKey
	return course
}

// getCourse builds and returns a new course from the provided arguments, no global state is changed
func getCourse(courseNum string, session schema.AcademicSession, rowInfo map[string]*goquery.Selection, classInfo map[string]string) *schema.Course {
	CoursePrefix, CourseNumber := getPrefixAndNumber(classInfo)

	course := schema.Course{
		Id:                     primitive.NewObjectID(),
		Course_number:          CourseNumber,
		Subject_prefix:         CoursePrefix,
		Title:                  utils.TrimWhitespace(rowInfo["Course Title:"].Text()),
		Description:            utils.TrimWhitespace(rowInfo["Description:"].Text()),
		School:                 utils.TrimWhitespace(rowInfo["College:"].Text()),
		Credit_hours:           classInfo["Semester Credit Hours:"],
		Class_level:            classInfo["Class Level:"],
		Activity_type:          classInfo["Activity Type:"],
		Grading:                classInfo["Grading:"],
		Internal_course_number: courseNum,
		Catalog_year:           getCatalogYear(session),
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

func getCatalogYear(session schema.AcademicSession) string {
	sessionYear, err := strconv.Atoi(session.Name[0:2])
	if err != nil {
		panic(err)
	}
	sessionSemester := session.Name[2]
	switch sessionSemester {
	case 'F':
		return strconv.Itoa(sessionYear)
	case 'S':
		return strconv.Itoa(sessionYear - 1)
	case 'U':
		return strconv.Itoa(sessionYear - 1)
	default:
		panic(fmt.Errorf("encountered invalid session semester '%c!'", sessionSemester))
	}
}

func getPrefixAndNumber(classInfo map[string]string) (string, string) {
	if sectionId, ok := classInfo["Class Section:"]; ok {
		// Get subject prefix and course number by doing a regexp match on the section id
		matches := coursePrefixRexp.FindStringSubmatch(sectionId)
		if len(matches) == 3 {
			return matches[1], matches[2]
		}
	}
	return "", ""
}
