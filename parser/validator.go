package parser

import (
	"log"
	"slices"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Main validation, putting everything together
func validate() {
	// Set up deferred handler for panics to display validation fails
	defer func() {
		if err := recover(); err != nil {
			log.Printf("VALIDATION FAILED: %s", err)
		}
	}()

	log.Printf("Validating courses...")
	courseKeys := utils.GetMapKeys(Courses)
	for i := range len(courseKeys) {
		course1 := Courses[courseKeys[i]]
		// Check for duplicate courses by comparing course_number, subject_prefix, and catalog_year as a compound key
		for j := i + 1; j < len(courseKeys); j++ {
			course2 := Courses[courseKeys[j]]
			valDuplicateCourses(course1, course2)
		}
		// Make sure course isn't referencing any nonexistent sections, and that course-section references are consistent both ways
		valCourseReference(course1, Sections)
	}
	courseKeys = nil
	log.Print("No invalid courses!")

	log.Print("Validating sections...")
	sectionKeys := utils.GetMapKeys(Sections)
	for i := range len(sectionKeys) {
		section1 := Sections[sectionKeys[i]]
		// Check for duplicate sections by comparing section_number, course_reference, and academic_session as a compound key
		for j := i + 1; j < len(sectionKeys); j++ {
			section2 := Sections[sectionKeys[j]]
			valDuplicateSections(section1, section2)
		}
		// Make sure section isn't referencing any nonexistent professors, and that section-professor references are consistent both ways
		valSectionReferenceProf(section1, Professors, ProfessorIDMap)

		// Make sure section isn't referencing a nonexistant course
		valSectionReferenceCourse(section1, CourseIDMap)
	}
	sectionKeys = nil
	log.Printf("No invalid sections!")

	log.Printf("Validating professors...")
	profKeys := utils.GetMapKeys(Professors)
	// Check for duplicate professors by comparing first_name, last_name, and sections as a compound key
	for i := range len(profKeys) {
		prof1 := Professors[profKeys[i]]
		for j := i + 1; j < len(profKeys); j++ {
			prof2 := Professors[profKeys[j]]
			valDuplicateProfs(prof1, prof2)
		}
	}
	log.Printf("No invalid professors!")
}

// Validate if the courses are duplicate
func valDuplicateCourses(course1 *schema.Course, course2 *schema.Course) {
	if course1.Catalog_year == course2.Catalog_year && course1.Course_number == course2.Course_number &&
		course1.Subject_prefix == course2.Subject_prefix {
		log.Printf("Duplicate course found for %s%s!", course1.Subject_prefix, course1.Course_number)
		log.Printf("Course 1: %v", course1)
		log.Printf("Course 2: %v", course2)
		log.Panic("Courses failed to validate!")
	}
}

// Validate course reference to sections
func valCourseReference(course *schema.Course, sections map[primitive.ObjectID]*schema.Section) {
	for _, sectionID := range course.Sections {
		section, exists := sections[sectionID]
		// validate if course references to some section not in the parsed sections
		if !exists {
			log.Printf("Nonexistent section reference found for %s%s!", course.Subject_prefix, course.Course_number)
			log.Printf("Referenced section ID: %s", sectionID)
			log.Printf("Course ID: %s", course.Id)
			log.Panic("Courses failed to validate!")
		}

		// validate if the ref sections references back to the course
		if section.Course_reference != course.Id {
			log.Printf("Inconsistent section reference found for %s%s! The course references the section, but not vice-versa!", course.Subject_prefix, course.Course_number)
			log.Printf("Referenced section ID: %s", sectionID)
			log.Printf("Course ID: %s", course.Id)
			log.Printf("Section course reference: %s", section.Course_reference)
			log.Panic("Courses failed to validate!")
		}
	}
}

// Validate if the sections are duplicate
func valDuplicateSections(section1 *schema.Section, section2 *schema.Section) {
	if section1.Section_number == section2.Section_number && section1.Course_reference == section2.Course_reference &&
		section1.Academic_session == section2.Academic_session {
		log.Print("Duplicate section found!")
		log.Printf("Section 1: %v", section1)
		log.Printf("Section 2: %v", section2)
		log.Panic("Sections failed to validate!")
	}
}

// Validate section reference to professor
func valSectionReferenceProf(section *schema.Section, profs map[string]*schema.Professor, profIDMap map[primitive.ObjectID]string) {
	for _, profID := range section.Professors {
		professorKey, exists := profIDMap[profID]
		// validate if the section references to some prof not in the parsed professors
		if !exists {
			log.Printf("Nonexistent professor reference found for section ID %s!", section.Id)
			log.Printf("Referenced professor ID: %s", profID)
			log.Panic("Sections failed to validate!")
		}

		// validate if the referenced professor references back to section
		if !slices.Contains(profs[professorKey].Sections, section.Id) {
			log.Printf("Inconsistent professor reference found for section ID %s! The section references the professor, but not vice-versa!", section.Id)
			log.Printf("Referenced professor ID: %s", profID)
			log.Panic("Sections failed to validate!")
		}
	}
}

// Validate section reference to course
func valSectionReferenceCourse(section *schema.Section, courseIDMap map[primitive.ObjectID]string) {
	_, exists := courseIDMap[section.Course_reference]
	// validate if section reference some course not in parsed courses
	if !exists {
		log.Printf("Nonexistent course reference found for section ID %s!", section.Id)
		log.Printf("Referenced course ID: %s", section.Course_reference)
		log.Panic("Sections failed to validate!")
	}
}

// Validate if the professors are duplicate
func valDuplicateProfs(prof1 *schema.Professor, prof2 *schema.Professor) {
	if prof1.First_name == prof2.First_name && prof1.Last_name == prof2.Last_name && prof1.Profile_uri == prof2.Profile_uri {
		log.Printf("Duplicate professor found!")
		log.Printf("Professor 1: %v", prof1)
		log.Printf("Professor 2: %v", prof2)
		log.Panic("Professors failed to validate!")
	}
}
