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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Globals for testing these validation units
var testCourses []*schema.Course
var testSections []*schema.Section
var testProfessors []*schema.Professor

// Map index of test sections to test courses
var indexMap map[int]int

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

	courseIndex := make(map[primitive.ObjectID]int)
	for i, course := range testCourses {
		courseIndex[course.Id] = i
	}

	indexMap = make(map[int]int, len(testSections))
	for i, section := range testSections {
		indexMap[i] = courseIndex[section.Course_reference]
	}
}

// TestDuplicateCoursesFail expects duplicates to trigger validation panic.
//
// Designed for fail cases
func TestDuplicateCoursesFail(t *testing.T) {
	for i := range len(testCourses) {
		t.Run(
			fmt.Sprintf("Duplicate course %v", i),
			func(t *testing.T) {
				testDuplicateFail("course", i, t)
			},
		)
	}
}

// TestDuplicateSectionsFail ensures duplicate sections are rejected.
//
// Designed for fail cases
func TestDuplicateSectionsFail(t *testing.T) {
	for i := range len(testSections) {
		t.Run(
			fmt.Sprintf("Duplicate section %v", i),
			func(t *testing.T) {
				testDuplicateFail("section", i, t)
			},
		)
	}
}

// TestDuplicateProfFail ensures duplicate professors fail validation.
//
// Designed for fail cases
func TestDuplicateProfFail(t *testing.T) {
	for i := range len(testProfessors) {
		t.Run(
			fmt.Sprintf("Duplicate professor %v", i),
			func(t *testing.T) {
				testDuplicateFail("professor", i, t)
			},
		)
	}
}

// TestDuplicateCoursesPass confirms unique courses validate successfully.
//
// Designed for pass case
func TestDuplicateCoursesPass(t *testing.T) {
	for i := range len(testCourses) {
		for j := i + 1; j < len(testCourses); j++ {
			t.Run(
				fmt.Sprintf("Duplicate courses %v, %v", i, j),
				func(t *testing.T) {
					testDuplicatePass("course", i, j, t)
				},
			)
		}
	}
}

// TestDuplicateSectionsPass confirms unique sections validate successfully.
//
// Designed for pass cases
func TestDuplicateSectionsPass(t *testing.T) {
	for i := range len(testSections) {
		for j := i + 1; j < len(testSections); j++ {
			t.Run(
				fmt.Sprintf("Duplicate sections %v, %v", i, j),
				func(t *testing.T) {
					testDuplicatePass("section", i, j, t)
				},
			)
		}
	}
}

// TestDuplicateProfPass confirms unique professors validate successfully.
//
// Designed for pass cases
func TestDuplicateProfPass(t *testing.T) {
	for i := range len(testProfessors) {
		for j := i + 1; j < len(testProfessors); j++ {
			t.Run(fmt.Sprintf("Duplicate professors %v, %v", i, j),
				func(t *testing.T) {
					testDuplicatePass("professor", i, j, t)
				},
			)
		}
	}
}

// TestCourseReferencePass ensures section references to courses succeed.
// Test if course references to anything nonexistent. Designed for pass case
func TestCourseReferencePass(t *testing.T) {
	sectionMap := make(map[primitive.ObjectID]*schema.Section)
	for _, section := range testSections {
		sectionMap[section.Id] = section
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
		valCourseReference(course, sectionMap)
	}
}

// Test if function log expected msgs when course references non-existent sections
// 2 types of fail:
//   - Course references non-existent section
//   - Section doesn't reference back to same course

// TestCourseReferenceFail1 detects missing course references during validation.
// This is fail: missing
func TestCourseReferenceFail1(t *testing.T) {
	for key, value := range indexMap {
		t.Run(
			fmt.Sprintf("Section %v & course %v", key, value),
			func(t *testing.T) {
				testCourseReferenceFail("missing", value, key, t)
			},
		)
	}
}

// TestCourseReferenceFail2 detects mismatched section-course references.
// This is fail: modified
func TestCourseReferenceFail2(t *testing.T) {
	for key, value := range indexMap {
		t.Run(
			fmt.Sprintf("Section %v & course %v", key, value),
			func(t *testing.T) {
				testCourseReferenceFail("modified", value, key, t)
			},
		)
	}
}

// TestSectionReferenceProfPass ensures section professor references are mutual.
// Test section reference to professor, designed for pass case
func TestSectionReferenceProfPass(t *testing.T) {
	// Build profIDMap & profs
	profIDMap := make(map[primitive.ObjectID]string)
	profs := make(map[string]*schema.Professor)

	for _, professor := range testProfessors {
		profIDMap[professor.Id] = professor.First_name + professor.Last_name
		profs[professor.First_name+professor.Last_name] = professor
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
		valSectionReferenceProf(section, profs, profIDMap)
	}
}

// TestSectionReferenceProfFail catches missing professor back-references.
// Test section reference to professors, designed for fail case
func TestSectionReferenceProfFail(t *testing.T) {

	profIDMap := make(map[primitive.ObjectID]string)
	profs := make(map[string]*schema.Professor)

	for i, professor := range testProfessors {
		if i != 0 {
			profIDMap[professor.Id] = professor.First_name + professor.Last_name
			profs[professor.First_name+professor.Last_name] = professor
		}
	}

	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	defer func() {
		logOutput := logBuffer.String()

		for _, msg := range []string{
			"Nonexistent professor reference found for section ID ObjectID(\"69cda9299bb0b8cd4cf4c732\")!",
			"Referenced professor ID: ObjectID(\"69cda9299bb0b8cd4cf4c734\")",
		} {
			if !strings.Contains(logOutput, msg) {
				t.Errorf("The function didn't log correct message. \nExpected \"%v\"\nGot \"%v\"", msg, logOutput)
			}
		}

		if r := recover(); r == nil {
			t.Errorf("The function didn't panic")
		} else {
			if r != "Sections failed to validate!" {
				t.Errorf("The function panic the wrong message")
			}
		}
	}()

	for _, section := range testSections {
		valSectionReferenceProf(section, profs, profIDMap)
	}
}

// TestSectionReferenceCourse verifies section-course reference validation.
// Test section reference to course
func TestSectionReferenceCourse(t *testing.T) {
	courseIDMap := make(map[primitive.ObjectID]string)
	for _, course := range testCourses {
		courseIDMap[course.Id] = course.Internal_course_number + course.Catalog_year
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
		valSectionReferenceCourse(section, courseIDMap)
	}
}

/******** BELOW HERE ARE HELPER FUNCTION FOR TESTS ABOVE ********/

// testDuplicateFail tests if validate() throws erros when encountering duplicate
// Design for fail cases
func testDuplicateFail(objType string, ix int, t *testing.T) {
	// the buffer used to capture the log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	// Determine the expected messages and panic messages based on object type
	var expectedMsgs []string
	var panicMsg string

	switch objType {
	case "course":
		failCourse := testCourses[ix]

		// list of msgs it must print
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
		logOutput := logBuffer.String() // log output after running the function

		// Log output needs to contain lines in the list
		for _, msg := range expectedMsgs {
			if !strings.Contains(logOutput, msg) {
				t.Errorf("Exptected the message for %v: %v", objType, msg)
			}
		}

		// Test whether func panics and sends the correct panic msg
		if r := recover(); r == nil {
			t.Errorf("The function didn't panic for %v", objType)
		} else {
			if r != panicMsg {
				// The panic msg is incorrect
				t.Errorf("The function outputted the wrong panic message for %v.", objType)
			}
		}
	}()

	// Run func
	switch objType {
	case "course":
		valDuplicateCourses(testCourses[ix], testCourses[ix])
	case "section":
		valDuplicateSections(testSections[ix], testSections[ix])
	case "professor":
		valDuplicateProfs(testProfessors[ix], testProfessors[ix])
	}
}

// testDuplicatePass tests if func doesn't log anything and doesn't panic.
//
// Design for pass cases
func testDuplicatePass(objType string, ix1 int, ix2 int, t *testing.T) {
	// Buffer to capture the output
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

	// Run func according to the object type.
	// Choose pair of objects which are not duplicate
	switch objType {
	case "course":
		valDuplicateCourses(testCourses[ix1], testCourses[ix2])
	case "section":
		valDuplicateSections(testSections[ix1], testSections[ix2])
	case "professor":
		valDuplicateProfs(testProfessors[ix1], testProfessors[ix2])
	}
}

// testCourseReferenceFail tests if func doesn't log anything and doesn't panic.
//
// fail = "missing" means it lacks one sections
// fail = "modified" means one section's course reference has been modified
func testCourseReferenceFail(fail string, courseIx int, sectionIx int, t *testing.T) {
	sectionMap := make(map[primitive.ObjectID]*schema.Section)

	var sectionID, originalID primitive.ObjectID // used to store IDs of modified sections

	// Build the failed section map based on fail type
	switch fail {
	case "missing":
		// Misses a section
		for i, section := range testSections {
			if sectionIx != i {
				sectionMap[section.Id] = section
			} else {
				sectionID = section.Id // Nonexistent ID referenced by course
			}
		}
	case "modified":
		// One section doesn't reference to correct courses
		for i, section := range testSections {
			sectionMap[section.Id] = section
			if sectionIx == i {
				// Save the section ID and original course reference to be restored later on
				sectionID = section.Id
				originalID = section.Course_reference

				// Modified part
				sectionMap[section.Id].Course_reference = primitive.NewObjectID()
			}
		}
	}

	// Expected msgs
	var expectedMsgs []string

	// The course that references nonexistent stuff
	var failCourse *schema.Course

	if fail == "missing" {
		failCourse = testCourses[courseIx]

		expectedMsgs = []string{
			fmt.Sprintf("Nonexistent section reference found for %v%v!", failCourse.Subject_prefix, failCourse.Course_number),
			fmt.Sprintf("Referenced section ID: %s\nCourse ID: %s", sectionID, failCourse.Id),
		}
	} else {
		failCourse = testCourses[courseIx]
		failSection := testSections[sectionIx]

		expectedMsgs = []string{
			fmt.Sprintf("Inconsistent section reference found for %v%v! The course references the section, but not vice-versa!",
				failCourse.Subject_prefix, failCourse.Course_number),
			fmt.Sprintf("Referenced section ID: %s\nCourse ID: %s\nSection course reference: %s",
				failSection.Id, failCourse.Id, failSection.Course_reference),
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
			sectionMap[sectionID].Course_reference = originalID
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
		valCourseReference(course, sectionMap)
	}
}
