package parser

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var coursePrefixRexp *regexp.Regexp = regexp.MustCompile(`^([A-Z]{2,4})([0-9V]{4})`)
var contactRegexp *regexp.Regexp = regexp.MustCompile(`\(([0-9]+)-([0-9]+)\)\s+([SUFY]+)`)

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

func parseCourse(courseNum string, session schema.AcademicSession, rowInfo map[string]string, classInfo map[string]string) *schema.Course {
	// Courses are internally keyed by their internal course number and the catalog year they're part of
	catalogYear := getCatalogYear(session)
	courseKey := courseNum + catalogYear

	// Don't recreate the course if it already exists
	course, courseExists := Courses[courseKey]
	if courseExists {
		return course
	}

	// Get subject prefix and course number by doing a regexp match on the section id
	sectionId := classInfo["Class Section:"]
	idMatches := coursePrefixRexp.FindStringSubmatch(sectionId)

	course = &schema.Course{}

	course.Id = schema.IdWrapper(primitive.NewObjectID().Hex())
	course.Course_number = idMatches[2]
	course.Subject_prefix = idMatches[1]
	course.Title = rowInfo["Course Title:"]
	course.Description = rowInfo["Description:"]
	course.School = rowInfo["College:"]
	course.Credit_hours = classInfo["Semester Credit Hours:"]
	course.Class_level = classInfo["Class Level:"]
	course.Activity_type = classInfo["Activity Type:"]
	course.Grading = classInfo["Grading:"]
	course.Internal_course_number = courseNum

	// Get closure for parsing course requisites (god help me)
	enrollmentReqs, hasEnrollmentReqs := rowInfo["Enrollment Reqs:"]
	ReqParsers[course.Id] = getReqParser(course, hasEnrollmentReqs, enrollmentReqs)

	// Try to get lecture/lab contact hours and offering frequency from course description
	contactMatches := contactRegexp.FindStringSubmatch(course.Description)
	// Length of contactMatches should be 4 upon successful match
	if len(contactMatches) == 4 {
		course.Lecture_contact_hours = contactMatches[1]
		course.Laboratory_contact_hours = contactMatches[2]
		course.Offering_frequency = contactMatches[3]
	}

	// Set the catalog year
	course.Catalog_year = catalogYear

	Courses[courseKey] = course
	CourseIDMap[course.Id] = courseKey
	return course
}
