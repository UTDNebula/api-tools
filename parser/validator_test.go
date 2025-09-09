package parser

import (
	"fmt"
	"testing"

	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Test duplicate courses. Designed for fail cases
func TestDuplicateCoursesFail(t *testing.T) {
	t.Parallel()

	for i, course := range testCourses {
		name := fmt.Sprintf("Duplicate course %v", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer FailTestIfNoPanic(t, name)
			valDuplicateCourses(&course, &course)
		})
	}
}

// Test duplicate sections. Designed for fail cases
func TestDuplicateSectionsFail(t *testing.T) {
	t.Parallel()

	for i, section := range testSections {
		name := fmt.Sprintf("Duplicate section %v", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer FailTestIfNoPanic(t, name)
			valDuplicateSections(&section, &section)
		})
	}
}

// Test duplicate professors . Designed for fail cases
func TestDuplicateProfFail(t *testing.T) {
	t.Parallel()

	for i, prof := range testProfessors {
		name := fmt.Sprintf("Duplicate professor %v", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer FailTestIfNoPanic(t, name)
			valDuplicateProfs(&prof, &prof)
		})
	}
}

// Test duplicate courses. Designed for pass case
func TestDuplicateCoursesPass(t *testing.T) {
	t.Parallel()

	last := len(testCourses) - 1
	for i, course := range testCourses {
		if i == last {
			break
		}
		name := fmt.Sprintf("Duplicate courses %v %v", i, i+1)
		next := testCourses[i+1]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer FailTestIfPanic(t, name)
			valDuplicateCourses(&course, &next)
		})
	}
}

// Test duplicate sections. Designed for pass cases
func TestDuplicateSectionsPass(t *testing.T) {
	t.Parallel()

	last := len(testSections) - 1
	for i, section := range testSections {
		if i == last {
			break
		}
		name := fmt.Sprintf("Duplicate sections %v %v", i, i+1)
		next := testSections[i+1]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer FailTestIfPanic(t, name)
			valDuplicateSections(&section, &next)
		})
	}
}

// Test duplicate professors. Designed for pass cases
func TestDuplicateProfPass(t *testing.T) {
	t.Parallel()

	last := len(testProfessors) - 1
	for i, prof := range testProfessors {
		if i == last {
			break
		}
		name := fmt.Sprintf("Duplicate sections %v %v", i, i+1)
		next := testProfessors[i+1]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer FailTestIfPanic(t, name)
			valDuplicateProfs(&prof, &next)
		})
	}
}

// Test if course references to anything nonexistent. Designed for pass case
func TestCourseReferencePass(t *testing.T) {
	t.Parallel()

	sectionMap := make(map[primitive.ObjectID]*schema.Section)
	for _, section := range testSections {
		sectionMap[section.Id] = &section
	}

	for i, course := range testCourses {
		name := fmt.Sprintf("Course Reference %d", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer FailTestIfPanic(t, name)
			valCourseReference(&course, sectionMap)
		})
	}
}

// Fail 1 : Course references non-existent section
func TestCourseReferenceFail1(t *testing.T) {
	t.Parallel()

	sectionMap := make(map[primitive.ObjectID]*schema.Section)
	for _, section := range testSections {
		sectionMap[section.Id] = &section
	}

	for i, course := range testCourses {
		name := fmt.Sprintf("Course Reference Fail Type 1 %d", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			courseCopy := course
			courseCopy.Sections = append(courseCopy.Sections, primitive.NewObjectID())

			defer FailTestIfNoPanic(t, name)
			valCourseReference(&courseCopy, sectionMap)
		})
	}
}

// Fail 2 : Section doesn't reference back to same course
func TestCourseReferenceFail2(t *testing.T) {
	t.Parallel()

	sectionMap := make(map[primitive.ObjectID]*schema.Section)
	for _, section := range testSections {
		sectionMap[section.Id] = &section
	}

	for i, course := range testCourses {

		name := fmt.Sprintf("Course Reference Fail Type 2 %d", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mapCopy := sectionMap

			for _, id := range course.Sections {
				mapCopy[id].Course_reference = primitive.NewObjectID()
			}

			defer FailTestIfNoPanic(t, name)
			valCourseReference(&course, mapCopy)
		})
	}
}

// Test section reference to professor, designed for pass case
func TestSectionReferenceProfPass(t *testing.T) {
	t.Parallel()

	// Build profIDMap & profs
	profIDMap := make(map[primitive.ObjectID]string)
	profs := make(map[string]*schema.Professor)

	for _, professor := range testProfessors {
		profIDMap[professor.Id] = professor.First_name + professor.Last_name
		profs[professor.First_name+professor.Last_name] = &professor
	}

	for i, section := range testSections {
		name := fmt.Sprintf("Section Reference %d", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer FailTestIfPanic(t, name)
			valSectionReferenceProf(&section, profs, profIDMap)
		})
	}
}

// Fail 1 : section references a professor not parsed (in the profIDMap)
func TestSectionReferenceProfFail1(t *testing.T) {
	t.Parallel()

	profIDMap := make(map[primitive.ObjectID]string)
	profs := make(map[string]*schema.Professor)

	for _, professor := range testProfessors {
		profIDMap[professor.Id] = professor.First_name + professor.Last_name
		profs[professor.First_name+professor.Last_name] = &professor
	}

	for i, section := range testSections {
		name := fmt.Sprintf("Section Reference Fail Type 1 %d", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sectionCopy := section
			sectionCopy.Professors = append(sectionCopy.Professors, primitive.NewObjectID())

			defer FailTestIfNoPanic(t, name)
			valSectionReferenceProf(&sectionCopy, profs, profIDMap)
		})
	}
}

// Fail 2 : referenced professor does not back to the section
func TestSectionReferenceProfFail2(t *testing.T) {
	t.Parallel()

	profIDMap := make(map[primitive.ObjectID]string)
	profs := make(map[string]*schema.Professor)

	for _, professor := range testProfessors {
		profIDMap[professor.Id] = professor.First_name + professor.Last_name
		profs[professor.First_name+professor.Last_name] = &professor
	}

	for i, section := range testSections {
		name := fmt.Sprintf("Section Reference Fail Type 2 %d", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// some edge case sections have no professor and therefore cant fail this case
			if len(section.Professors) == 0 {
				return
			}

			key := profIDMap[section.Professors[0]]
			profCopy := profs[key]
			// remove all sections
			profCopy.Sections = []primitive.ObjectID{}

			profsCopy := map[string]*schema.Professor{key: profCopy}
			defer FailTestIfNoPanic(t, name)
			valSectionReferenceProf(&section, profsCopy, profIDMap)
		})
	}
}

// Test section reference to course
func TestSectionReferenceCoursePass(t *testing.T) {
	t.Parallel()

	courseIDMap := make(map[primitive.ObjectID]string)
	for _, course := range testCourses {
		courseIDMap[course.Id] = course.Internal_course_number + course.Catalog_year
	}

	for i, section := range testSections {
		name := fmt.Sprintf("Section Reference Course %d", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer FailTestIfPanic(t, name)
			valSectionReferenceCourse(&section, courseIDMap)
		})
	}
}

// Fail: Section refers to a course that does not exist
func TestSectionReferenceCourseFail(t *testing.T) {
	t.Parallel()

	courseIDMap := make(map[primitive.ObjectID]string)

	for i, section := range testSections {
		name := fmt.Sprintf("Section Reference Course %d", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			defer FailTestIfNoPanic(t, name)
			valSectionReferenceCourse(&section, courseIDMap)
		})
	}
}
