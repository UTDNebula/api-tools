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

// Map professor to sections
var profSectionMap map[int][]int

// Map sections to professor
var sectionProfMap map[int][]int

// Map courses to sections
var courseSectionMap map[int][]int

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

	// Mapping between professor and sections
	profSectionMap = map[int][]int{0: {4, 5}, 1: {4, 5}, 2:{0}, 3:{1}}

	// Reverse mappings
	courseSectionMap = map[int][]int{}
	for sectionIndex, courseIndex := range sectionCourseMap { 
		courseSectionMap[courseIndex] = append(courseSectionMap[courseIndex], sectionIndex) 
	}

	sectionProfMap = map[int][]int{}
	for profIndex, sections := range profSectionMap { 
		for _, sectionIndex := range sections { 
			sectionProfMap[sectionIndex] = append(sectionProfMap[sectionIndex], profIndex) 
		} 
	}

	// Set up keys for courses
	for i := range testCourses {
		testCourses[i].Key = schema.CourseKey {
			Course_number: testCourses[i].Course_number,
			Catalog_year: testCourses[i].Catalog_year,
			Subject_prefix: testCourses[i].Subject_prefix,
		}

	sectionKeys := []schema.SectionKey{} 
	for _, sectionIndex := range courseSectionMap[i] {
			sectionKey := schema.SectionKey {
				Section_number: testSections[sectionIndex].Section_number,
				Term: testSections[sectionIndex].Academic_session.Name,
				Course_number: testCourses[i].Course_number,
				Catalog_year: testCourses[i].Catalog_year,
				Subject_prefix: testCourses[i].Subject_prefix, 
			}
			sectionKeys = append(sectionKeys, sectionKey) 
		}
		testCourses[i].Section_keys = sectionKeys
	}

	// Set up keys for sections
	for i := range testSections {
		testSections[i].Key = schema.SectionKey {
			Section_number: testSections[i].Section_number,
			Term: testSections[i].Academic_session.Name,
			Course_number: testCourses[sectionCourseMap[i]].Course_number,
			Catalog_year: testCourses[sectionCourseMap[i]].Catalog_year,
			Subject_prefix: testCourses[sectionCourseMap[i]].Subject_prefix,
		}

		testSections[i].Course_key = schema.CourseKey {
			Course_number: testCourses[sectionCourseMap[i]].Course_number,
			Catalog_year: testCourses[sectionCourseMap[i]].Catalog_year,
			Subject_prefix: testCourses[sectionCourseMap[i]].Subject_prefix,
		}

		professorKeys := []schema.ProfessorKey{}
		for _, professorIndex := range sectionProfMap[i] {
			professorKey := schema.ProfessorKey {
				First_name: testProfessors[professorIndex].First_name,
				Last_name: testProfessors[professorIndex].Last_name,
			}
			professorKeys = append(professorKeys, professorKey)
		}
		testSections[i].Professor_keys = professorKeys
	}

	// Set up keys for professors
	for i := range testProfessors {
		testProfessors[i].Key = schema.ProfessorKey {
			First_name: testProfessors[i].First_name,
			Last_name: testProfessors[i].Last_name,
		}
		
		sectionKeys := []schema.SectionKey{}
		for _, sectionIndex := range profSectionMap[i] {
			sectionKey := schema.SectionKey {
				Section_number: testSections[sectionIndex].Section_number,
				Term: testSections[sectionIndex].Academic_session.Name,
				Course_number: testCourses[sectionCourseMap[sectionIndex]].Course_number,
				Catalog_year: testCourses[sectionCourseMap[sectionIndex]].Catalog_year,
				Subject_prefix: testCourses[sectionCourseMap[sectionIndex]].Subject_prefix,
			}
			sectionKeys = append(sectionKeys, sectionKey)
		}
		testProfessors[i].Section_keys = sectionKeys
	}
}

// Test duplicate courses. Designed for fail cases
// TestDuplicateCoursesFail expects duplicates to trigger validation panic.
func TestDuplicateCoursesFail(t *testing.T) {
	for i := range len(testCourses) {
		t.Run(fmt.Sprintf("Duplicate course %v", i), func(t *testing.T) {
			testDuplicateFail("course", i, t)
		})
	}
}

// Test duplicate sections. Designed for fail cases
// TestDuplicateSectionsFail ensures duplicate sections are rejected.
func TestDuplicateSectionsFail(t *testing.T) {
	for i := range len(testSections) {
		t.Run(fmt.Sprintf("Duplicate section %v", i), func(t *testing.T) {
			testDuplicateFail("section", i, t)
		})
	}
}

// Test duplicate professors . Designed for fail cases
// TestDuplicateProfFail ensures duplicate professors fail validation.
func TestDuplicateProfFail(t *testing.T) {
	for i := range len(testProfessors) {
		t.Run(fmt.Sprintf("Duplicate professor %v", i), func(t *testing.T) {
			testDuplicateFail("professor", i, t)
		})
	}
}

// Test duplicate courses. Designed for pass case
// TestDuplicateCoursesPass confirms unique courses validate successfully.
func TestDuplicateCoursesPass(t *testing.T) {
	for i := range len(testCourses) - 1 {
		t.Run(fmt.Sprintf("Duplicate courses %v, %v", i, i+1), func(t *testing.T) {
			testDuplicatePass("course", i, i+1, t)
		})
	}
}

// Test duplicate sections. Designed for pass cases
// TestDuplicateSectionsPass confirms unique sections validate successfully.
func TestDuplicateSectionsPass(t *testing.T) {
	for i := range len(testSections) - 1 {
		t.Run(fmt.Sprintf("Duplicate sections %v, %v", i, i+1), func(t *testing.T) {
			testDuplicatePass("section", i, i+1, t)
		})
	}
}

// Test duplicate professors. Designed for pass cases
// TestDuplicateProfPass confirms unique professors validate successfully.
func TestDuplicateProfPass(t *testing.T) {
	for i := range len(testProfessors) - 1 {
		t.Run(fmt.Sprintf("Duplicate professors %v, %v", i, i+1), func(t *testing.T) {
			testDuplicatePass("professor", i, i+1, t)
		})
	}
}

// Test if course references to anything nonexistent. Designed for pass case
// TestCourseReferencePass ensures section references to courses succeed.
func TestCourseReferencePass(t *testing.T) {
	sectionMap := make(map[schema.SectionKey]*schema.Section)
	for _, section := range testSections {
		sectionMap[section.Key] = section
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
//
// This is fail: missing
// Legacy test, no longer needed?
// TestCourseReferenceFail1 detects missing course references during validation.
// func TestCourseReferenceFail1(t *testing.T) {
// 	for key, value := range indexMap {
// 		t.Run(fmt.Sprintf("Section %v & course %v", key, value), func(t *testing.T) {
// 			testCourseReferenceFail("missing", value, key, t)
// 		})
// 	}
// }

// This is fail: modified
// TestCourseReferenceFail2 detects mismatched section-course references.
//Legacy test, no longer needed?
// func TestCourseReferenceFail2(t *testing.T) {
// 	for key, value := range indexMap {
// 		t.Run(fmt.Sprintf("Section %v & course %v", key, value), func(t *testing.T) {
// 			testCourseReferenceFail("modified", value, key, t)
// 		})
// 	}
// }

// Test section reference to professor, designed for pass case
// TestSectionReferenceProfPass ensures section professor references are mutual.
func TestSectionReferenceProfPass(t *testing.T) {
	// Build profs maps
	profs := make(map[schema.ProfessorKey]*schema.Professor)

	for _, professor := range testProfessors {
		profKey := schema.ProfessorKey {
			First_name: professor.First_name,
			Last_name: professor.Last_name,
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

// Test section reference to professors, designed for fail case
// TestSectionReferenceProfFail catches missing professor back-references.
func TestSectionReferenceProfFail(t *testing.T) {
	profs := make(map[schema.ProfessorKey]*schema.Professor)

	for i, professor := range testProfessors {
		if i != 0 {
			profKey := schema.ProfessorKey {
				First_name: professor.First_name,
				Last_name: professor.Last_name,
			}
			profs[profKey] = professor
		}
	}

	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	defer func() {
		logOutput := logBuffer.String()
		for _, msg := range []string{
			"Nonexistent professor reference found for section ID ObjectID(\"67d07ee0c972c18731e23bea\")!",
			"Referenced professor key: {Naim Bugra Ozel}",
		} {
			if !strings.Contains(logOutput, msg) {
				t.Errorf("The function didn't log correct message. Expected \"%v\"", msg)
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
		valSectionReferenceProf(section, profs)
	}
}

// Test section reference to course
// TestSectionReferenceCourse verifies section-course reference validation.
func TestSectionReferenceCourse(t *testing.T) {
	coursesByKey := make(map[schema.CourseKey]*schema.Course)
	for _, course := range testCourses {
		coursesByKey[course.Key] = course
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

/******** BELOW HERE ARE HELPER FUNCTION FOR TESTS ABOVE ********/

// Test if validate() throws erros when encountering duplicate
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

// Test if func doesn't log anything and doesn't panic.
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

// fail = "missing" means it lacks one sections
// fail = "modified" means one section's course reference has been modified
// For course-section, no longer needed
// func testCourseReferenceFail(fail string, courseIx int, sectionIx int, t *testing.T) {
// 	sectionMap := make(map[primitive.ObjectID]*schema.Section)

// 	var sectionID, originalID primitive.ObjectID // used to store IDs of modified sections

// 	// Build the failed section map based on fail type
// 	switch fail {
// 	case "missing":
// 		// Misses a section
// 		for i, section := range testSections {
// 			if sectionIx != i {
// 				sectionMap[section.Id] = section
// 			} else {
// 				sectionID = section.Id // Nonexistent ID referenced by course
// 			}
// 		}
// 	case "modified":
// 		// One section doesn't reference to correct courses
// 		for i, section := range testSections {
// 			sectionMap[section.Id] = section
// 			if sectionIx == i {
// 				// Save the section ID and original course reference to be restored later on
// 				sectionID = section.Id
// 				originalID = section.Course_reference

// 				// Modified part
// 				sectionMap[section.Id].Course_reference = primitive.NewObjectID()
// 			}
// 		}
// 	}

// 	// Expected msgs
// 	var expectedMsgs []string

// 	// The course that references nonexistent stuff
// 	var failCourse *schema.Course

// 	if fail == "missing" {
// 		failCourse = testCourses[courseIx]

// 		expectedMsgs = []string{
// 			fmt.Sprintf("Nonexistent section reference found for %v%v!", failCourse.Subject_prefix, failCourse.Course_number),
// 			fmt.Sprintf("Referenced section ID: %s\nCourse ID: %s", sectionID, failCourse.Id),
// 		}
// 	} else {
// 		failCourse = testCourses[courseIx]
// 		failSection := testSections[sectionIx]

// 		expectedMsgs = []string{
// 			fmt.Sprintf("Inconsistent section reference found for %v%v! The course references the section, but not vice-versa!",
// 				failCourse.Subject_prefix, failCourse.Course_number),
// 			fmt.Sprintf("Referenced section ID: %s\nCourse ID: %s\nSection course reference: %s",
// 				failSection.Id, failCourse.Id, failSection.Course_reference),
// 		}
// 	}

// 	// Buffer to capture the output
// 	var logBuffer bytes.Buffer
// 	log.SetOutput(&logBuffer)

// 	defer func() {
// 		logOutput := logBuffer.String()

// 		for _, msg := range expectedMsgs {
// 			if !strings.Contains(logOutput, msg) {
// 				t.Errorf("The function didn't log correct message. Expected \"%v\"", msg)
// 			}
// 		}

// 		// Restore to original course reference of modified section (if needed)
// 		if fail == "modified" {
// 			sectionMap[sectionID].Course_reference = originalID
// 		}

// 		if r := recover(); r == nil {
// 			t.Errorf("The function didn't panic")
// 		} else {
// 			if r != "Courses failed to validate!" {
// 				t.Errorf("The function panic the wrong message")
// 			}
// 		}
// 	}()

// 	// Run func
// 	for _, course := range testCourses {
// 		valCourseReference(course, sectionMap)
// 	}
// }