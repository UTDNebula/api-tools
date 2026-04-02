package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/UTDNebula/nebula-api/api/schema"
)

// Globals for testing these validation units
var testCourses []*schema.Course
var testSections []*schema.Section
var testProfessors []*schema.Professor

// Map index of test sections to test courses
var sectionCourseMap map[int]int

func init() {
	// Parse the test courses
	data, err := os.ReadFile("./testdata/courses.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &testCourses)
	if err != nil {
		panic(err)
	}

	// Parse the test sections
	data, err = os.ReadFile("./testdata/sections.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &testSections)
	if err != nil {
		panic(err)
	}

	// Parse the test professors
	data, err = os.ReadFile("./testdata/professors.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &testProfessors)
	if err != nil {
		panic(err)
	}

	// The correct mapping between sections and courses
	sectionCourseMap = map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 4: 4, 5: 4}
}

// Test duplicate courses. Designed for fail cases
func TestDuplicateCoursesFail(t *testing.T) {
	for i := range len(testCourses) {
		t.Run(fmt.Sprintf("Duplicate course %v", i), func(t *testing.T) {
			testDuplicateFail("course", i, t)
		})
	}
}

// Test duplicate sections. Designed for fail cases
func TestDuplicateSectionsFail(t *testing.T) {
	for i := range len(testSections) {
		t.Run(fmt.Sprintf("Duplicate section %v", i), func(t *testing.T) {
			testDuplicateFail("section", i, t)
		})
	}
}

// Test duplicate professors . Designed for fail cases
func TestDuplicateProfFail(t *testing.T) {
	for i := range len(testProfessors) {
		t.Run(fmt.Sprintf("Duplicate professor %v", i), func(t *testing.T) {
			testDuplicateFail("professor", i, t)
		})
	}
}

// Test duplicate courses. Designed for pass case
func TestDuplicateCoursesPass(t *testing.T) {
	for i := range len(testCourses) - 1 {
		t.Run(fmt.Sprintf("Duplicate courses %v, %v", i, i+1), func(t *testing.T) {
			testDuplicatePass("course", i, i+1, t)
		})
	}
}

// Test duplicate sections. Designed for pass cases
func TestDuplicateSectionsPass(t *testing.T) {
	for i := range len(testSections) - 1 {
		t.Run(fmt.Sprintf("Duplicate sections %v, %v", i, i+1), func(t *testing.T) {
			testDuplicatePass("section", i, i+1, t)
		})
	}
}

// Test duplicate professors. Designed for pass cases
func TestDuplicateProfPass(t *testing.T) {
	for i := range len(testProfessors) - 1 {
		t.Run(fmt.Sprintf("Duplicate professors %v, %v", i, i+1), func(t *testing.T) {
			testDuplicatePass("professor", i, i+1, t)
		})
	}
}

// Test if course references to anything nonexistent. Designed for pass case
func TestCourseReferencePass(t *testing.T) {
	courseSectionMap := make(map[schema.CourseKey]map[schema.SectionKey]*schema.Section)
	for _, section := range testSections {
		courseKey := section.Course

		if courseSectionMap[courseKey] == nil {
			courseSectionMap[courseKey] = make(map[schema.SectionKey]*schema.Section)
		}

		sectionKey := schema.SectionKey{
			Subject_prefix: section.Course.Subject_prefix,
			Course_number:  section.Course.Course_number,
			Catalog_year:   section.Course.Catalog_year,
			Section_number: section.Section_number,
			Term:           section.Academic_session.Name,
		}

		courseSectionMap[courseKey][sectionKey] = section
	}

	// Buffer to capture the output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	defer func() {
		logOutput := logBuffer.String()

		if logOutput != "" {
			t.Errorf("Expected nothing printed in log")
		}
		if r := recover(); r != nil {
			t.Errorf("The function panic unexpectedly for course")
		}
	}()

	// Run func
	for _, course := range testCourses {
		valCourseReference(course, courseSectionMap)
	}
}

// Test if function log expected msgs when course references non-existent sections
// 2 types of fail:
//   - Course references non-existent section
//   - Section doesn't reference back to same course

// This is fail: missing
func TestCourseReferenceFail1(t *testing.T) {
	for key, value := range sectionCourseMap {
		t.Run(fmt.Sprintf("Section %v & course %v", key, value), func(t *testing.T) {
			testCourseReferenceFail("missing", value, key, t)
		})
	}
}

// This is fail: modified
func TestCourseReferenceFail2(t *testing.T) {
	for key, value := range sectionCourseMap {
		t.Run(fmt.Sprintf("Section %v & course %v", key, value), func(t *testing.T) {
			testCourseReferenceFail("modified", value, key, t)
		})
	}
}

// Test section reference to professor, designed for pass case
func TestSectionReferenceProfPass(t *testing.T) {
	profs := make(map[schema.ProfessorKey]*schema.Professor)

	for _, professor := range testProfessors {
		profKey := schema.ProfessorKey{
			First_name: professor.First_name,
			Last_name:  professor.Last_name,
		}
		profs[profKey] = professor
	}

	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	defer func() {
		logOutput := logBuffer.String()
		if logOutput != "" {
			t.Errorf("Expected nothing printed in log")
		}
		if r := recover(); r != nil {
			t.Errorf("The function panic unexpectedly for section")
		}
	}()

	for _, section := range testSections {
		valSectionReferenceProf(section, profs)
	}
}

// Test section reference to course
func TestSectionReferenceCourse(t *testing.T) {
	coursesByKey := make(map[schema.CourseKey]*schema.Course)
	for _, course := range testCourses {
		courseKey := schema.CourseKey{
			Subject_prefix: course.Subject_prefix,
			Course_number:  course.Course_number,
			Catalog_year:   course.Catalog_year,
		}
		coursesByKey[courseKey] = course
	}

	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	defer func() {
		logOutput := logBuffer.String()
		if logOutput != "" {
			t.Errorf("Expected nothing printed in log")
		}
		if r := recover(); r != nil {
			t.Errorf("The function panic unexpectedly for section")
		}
	}()

	for _, section := range testSections {
		valSectionReferenceCourse(section, coursesByKey)
	}
}

// Test if function log expected msgs when course references section with mismatched compound key
// This is fail: wrong key
func TestCourseReferenceFail3(t *testing.T) {
	for key, value := range sectionCourseMap {
		t.Run(fmt.Sprintf("Section %v & course %v", key, value), func(t *testing.T) {
			testCourseReferenceFail("wrongkey", value, key, t)
		})
	}
}

/******** BELOW HERE ARE HELPER FUNCTION FOR TESTS ABOVE ********/

// Test if validate() throws errors when encountering duplicate
// Designed for fail cases
func testDuplicateFail(objType string, ix int, t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	var expectedMsgs []string
	var panicMsg string

	switch objType {
	case "course":
		failCourse := testCourses[ix]

		expectedMsgs = []string{
			fmt.Sprintf("Duplicate course found for %s%s!", failCourse.Subject_prefix, failCourse.Course_number),
			fmt.Sprintf("Course 1: %v\n\nCourse 2: %v", failCourse, failCourse),
		}
		panicMsg = "Courses failed to validate!"

	case "section":
		failSection := testSections[ix]

		expectedMsgs = []string{
			"Duplicate section found!",
			fmt.Sprintf("Section 1: %v\n\nSection 2: %v", failSection, failSection),
		}
		panicMsg = "Sections failed to validate!"

	case "professor":
		failProf := testProfessors[ix]

		expectedMsgs = []string{
			"Duplicate professor found!",
			fmt.Sprintf("Professor 1: %v\n\nProfessor 2: %v", failProf, failProf),
		}
		panicMsg = "Professors failed to validate!"
	}

	defer func() {
		logOutput := logBuffer.String()

		for _, msg := range expectedMsgs {
			if !strings.Contains(logOutput, msg) {
				t.Errorf("Expected the message for %v: %v", objType, msg)
			}
		}

		if r := recover(); r == nil {
			t.Errorf("The function didn't panic for %v", objType)
		} else if r != panicMsg {
			t.Errorf("The function outputted the wrong panic message for %v.", objType)
		}
	}()

	switch objType {
	case "course":
		valDuplicateCourses(testCourses[ix], testCourses[ix])
	case "section":
		valDuplicateSections(testSections[ix], testSections[ix])
	case "professor":
		valDuplicateProfs(testProfessors[ix], testProfessors[ix])
	}
}

// Test if func doesn't log anything and doesn't panic.
// Designed for pass cases
func testDuplicatePass(objType string, ix1 int, ix2 int, t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	defer func() {
		logOutput := logBuffer.String()
		if logOutput != "" {
			t.Errorf("Expected nothing in log for %s", objType)
		}
		if r := recover(); r != nil {
			t.Errorf("The function panic unexpectedly for %s", objType)
		}
	}()

	switch objType {
	case "course":
		valDuplicateCourses(testCourses[ix1], testCourses[ix2])
	case "section":
		valDuplicateSections(testSections[ix1], testSections[ix2])
	case "professor":
		valDuplicateProfs(testProfessors[ix1], testProfessors[ix2])
	}
}

// fail = "missing" means it lacks one sections
// fail = "modified" means one section's course reference has been modified
// fail = "wrongkey" means one section is stored under the wrong compound key
func testCourseReferenceFail(fail string, courseIx int, sectionIx int, t *testing.T) {
	courseSectionMap := make(map[schema.CourseKey]map[schema.SectionKey]*schema.Section)

	// Used to store keys of modified sections
	var sectionRef schema.SectionKey
	var actualSectionKey schema.SectionKey
	var sectionCourseRef, originalCourse schema.CourseKey

	// Build the failed section map based on fail type
	switch fail {
	case "missing":
		// Misses a section
		for i, section := range testSections {
			courseKey := section.Course
			if courseSectionMap[courseKey] == nil {
				courseSectionMap[courseKey] = make(map[schema.SectionKey]*schema.Section)
			}

			sectionKey := schema.SectionKey{
				Subject_prefix: section.Course.Subject_prefix,
				Course_number:  section.Course.Course_number,
				Catalog_year:   section.Course.Catalog_year,
				Section_number: section.Section_number,
				Term:           section.Academic_session.Name,
			}

			if sectionIx != i {
				courseSectionMap[courseKey][sectionKey] = section
			} else {
				sectionRef = sectionKey // Nonexistent key referenced by course
			}
		}

	case "modified":
		// One section doesn't reference to correct courses
		for i, section := range testSections {
			courseKey := section.Course
			if courseSectionMap[courseKey] == nil {
				courseSectionMap[courseKey] = make(map[schema.SectionKey]*schema.Section)
			}

			sectionKey := schema.SectionKey{
				Subject_prefix: section.Course.Subject_prefix,
				Course_number:  section.Course.Course_number,
				Catalog_year:   section.Course.Catalog_year,
				Section_number: section.Section_number,
				Term:           section.Academic_session.Name,
			}
			courseSectionMap[courseKey][sectionKey] = section

			if sectionIx == i {
				// Save the section ID and original course reference to be restored later on
				sectionRef = sectionKey
				sectionCourseRef = section.Course
				originalCourse = courseKey

				// Modified part
				courseSectionMap[courseKey][sectionKey].Course = schema.CourseKey{}
			}
		}

	case "wrongkey":
		// One section exists, but is stored under the wrong compound key
		// and the course references that same wrong key
		for i, section := range testSections {
			courseKey := section.Course
			if courseSectionMap[courseKey] == nil {
				courseSectionMap[courseKey] = make(map[schema.SectionKey]*schema.Section)
			}

			realSectionKey := schema.SectionKey{
				Subject_prefix: section.Course.Subject_prefix,
				Course_number:  section.Course.Course_number,
				Catalog_year:   section.Course.Catalog_year,
				Section_number: section.Section_number,
				Term:           section.Academic_session.Name,
			}

			if sectionIx == i {
				actualSectionKey = realSectionKey

				sectionRef = schema.SectionKey{
					Subject_prefix: section.Course.Subject_prefix,
					Course_number:  section.Course.Course_number,
					Catalog_year:   section.Course.Catalog_year,
					Section_number: section.Section_number,
					Term:           section.Academic_session.Name + "_WRONG",
				}

				// store section under wrong key so lookup succeeds
				courseSectionMap[courseKey][sectionRef] = section

				// replace the matching course reference with the wrong key
				course := testCourses[courseIx]
				for j, key := range course.Sections {
					if key == realSectionKey {
						course.Sections[j] = sectionRef
						break
					}
				}
			} else {
				courseSectionMap[courseKey][realSectionKey] = section
			}
		}
	}

	// Expected msgs
	var expectedMsgs []string

	// The course that references nonexistent stuff
	failCourse := testCourses[courseIx]
	failCourseKey := schema.CourseKey{
		Subject_prefix: failCourse.Subject_prefix,
		Course_number:  failCourse.Course_number,
		Catalog_year:   failCourse.Catalog_year,
	}

	if fail == "missing" {
		expectedMsgs = []string{
			fmt.Sprintf("Nonexistent section reference found for %s%s!", failCourse.Subject_prefix, failCourse.Course_number),
			fmt.Sprintf("Referenced section key: %+v\nCourse key: %+v", sectionRef, failCourseKey),
		}
	} else if fail == "modified" {
		failSection := testSections[sectionIx]

		expectedMsgs = []string{
			fmt.Sprintf("Inconsistent section reference found for %s%s! The course references the section, but not vice-versa!",
				failCourse.Subject_prefix, failCourse.Course_number),
			fmt.Sprintf("Referenced section key: %+v\nCourse key: %+v\nSection's course key: %+v",
				sectionRef, failCourseKey, failSection.Course),
		}
	} else {
		expectedMsgs = []string{
			fmt.Sprintf("Mismatched section key found for %s%s!", failCourse.Subject_prefix, failCourse.Course_number),
			fmt.Sprintf("Course stored section key: %+v\nActual section key: %+v",
				sectionRef, actualSectionKey),
		}
	}

	// Buffer to capture the output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	defer func() {
		logOutput := logBuffer.String()

		for _, msg := range expectedMsgs {
			if !strings.Contains(logOutput, msg) {
				t.Errorf("The function didn't log correct message. Expected \"%v\"", msg)
			}
		}

		// Restore to original course reference of modified section (if needed)
		if fail == "modified" {
			courseSectionMap[originalCourse][sectionRef].Course = sectionCourseRef
		}

		// Restore to original section key in course reference (if needed)
		if fail == "wrongkey" {
			failCourse := testCourses[courseIx]
			for j, key := range failCourse.Sections {
				if key == sectionRef {
					failCourse.Sections[j] = actualSectionKey
					break
				}
			}
		}

		if r := recover(); r == nil {
			t.Errorf("The function didn't panic")
		} else {
			if r != "Courses failed to validate!" {
				t.Errorf("The function panic the wrong message")
			}
		}
	}()
	// Run func
	for _, course := range testCourses {
		valCourseReference(course, courseSectionMap)
	}
}
