package parser

import (
	"log"
	"slices"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

// Main validation, putting everything together
func validate() {
	// Set up deferred handler for panics to display validation fails
	defer func() {
		if err := recover(); err != nil {
			log.Printf("VALIDATION FAILED: %s", err)
		}
	}()

	// Build maps for quick lookup by key
	courseSectionsByKey := make(map[schema.CourseKey]map[schema.SectionKey]*schema.Section)
	for _, section := range Sections {
		courseKey := section.Course
		if courseSectionsByKey[courseKey] == nil {
			courseSectionsByKey[courseKey] = make(map[schema.SectionKey]*schema.Section)
		}

		sectionKey := schema.SectionKey{
			Subject_prefix: section.Course.Subject_prefix,
			Course_number:  section.Course.Course_number,
			Catalog_year:   section.Course.Catalog_year,
			Section_number: section.Section_number,
			Term:           section.Academic_session.Name,
		}
		courseSectionsByKey[courseKey][sectionKey] = section
	}

	courseByKey := make(map[schema.CourseKey]*schema.Course)
	for _, course := range Courses {
		courseKey := schema.CourseKey{
			Subject_prefix: course.Subject_prefix,
			Course_number:  course.Course_number,
			Catalog_year:   course.Catalog_year,
		}
		courseByKey[courseKey] = course
	}

	profByKey := make(map[schema.ProfessorKey]*schema.Professor)
	for _, professor := range Professors {
		professorKey := schema.ProfessorKey{
			First_name: professor.First_name,
			Last_name:  professor.Last_name,
		}
		profByKey[professorKey] = professor
	}

	log.Printf("\nValidating courses...")
	courseKeys := utils.GetMapKeys(Courses)
	for i := range len(courseKeys) {
		course1 := Courses[courseKeys[i]]

		// Check for duplicate courses by comparing course_number, subject_prefix, and catalog_year as a compound key
		for j := i + 1; j < len(courseKeys); j++ {
			course2 := Courses[courseKeys[j]]
			valDuplicateCourses(course1, course2)
		}

		// Make sure course isn't referencing any nonexistent sections, and that course-section references are consistent both ways
		valCourseReference(course1, courseSectionsByKey)
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
		valSectionReferenceProf(section1, profByKey)

		// Make sure section isn't referencing a nonexistent course
		valSectionReferenceCourse(section1, courseByKey)
	}
	sectionKeys = nil
	log.Printf("No invalid sections!")

	log.Printf("Validating professors...")
	profKeys := utils.GetMapKeys(Professors)

	// Check for duplicate professors by comparing first_name, last_name, and profile_uri as a compound key
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
	if course1.Subject_prefix == course2.Subject_prefix &&
		course1.Course_number == course2.Course_number &&
		course1.Catalog_year == course2.Catalog_year {
		log.Printf("Duplicate course found for %s%s!", course1.Subject_prefix, course1.Course_number)
		log.Printf("Course 1: %v\n\nCourse 2: %v", course1, course2)
		log.Panic("Courses failed to validate!")
	}
}

// Validate course reference to sections
func valCourseReference(course *schema.Course, courseSections map[schema.CourseKey]map[schema.SectionKey]*schema.Section) {
	courseKey := schema.CourseKey{
		Subject_prefix: course.Subject_prefix,
		Course_number:  course.Course_number,
		Catalog_year:   course.Catalog_year,
	}

	sections := courseSections[courseKey]

	for _, sectionKey := range course.Sections {
		section, exists := sections[sectionKey]

		// validate if course references some section not in the parsed sections
		if !exists {
			log.Printf("Nonexistent section reference found for %s%s!", course.Subject_prefix, course.Course_number)
			log.Printf("Referenced section key: %+v\nCourse key: %+v", sectionKey, courseKey)
			log.Panic("Courses failed to validate!")
		}

		// validate if the referenced section points back to the same course
		if section.Course != courseKey {
			log.Printf("Inconsistent section reference found for %s%s! The course references the section, but not vice-versa!", course.Subject_prefix, course.Course_number)
			log.Printf("Referenced section key: %+v\nCourse key: %+v\nSection's course key: %+v", sectionKey, courseKey, section.Course)
			log.Panic("Courses failed to validate!")
		}

		// validate if the referenced section's own full compound key matches the key stored on the course
		expectedSectionKey := schema.SectionKey{
			Subject_prefix: section.Course.Subject_prefix,
			Course_number:  section.Course.Course_number,
			Catalog_year:   section.Course.Catalog_year,
			Section_number: section.Section_number,
			Term:           section.Academic_session.Name,
		}

		if expectedSectionKey != sectionKey {
			log.Printf("Mismatched section key found for %s%s!", course.Subject_prefix, course.Course_number)
			log.Printf("Course stored section key: %+v\nActual section key: %+v", sectionKey, expectedSectionKey)
			log.Panic("Courses failed to validate!")
		}
	}
}

// Validate if the sections are duplicate
func valDuplicateSections(section1 *schema.Section, section2 *schema.Section) {
	if section1.Section_number == section2.Section_number &&
		section1.Course == section2.Course &&
		section1.Academic_session == section2.Academic_session {
		log.Print("Duplicate section found!")
		log.Printf("Section 1: %v\n\nSection 2: %v", section1, section2)
		log.Panic("Sections failed to validate!")
	}
}

// Validate section reference to professor
func valSectionReferenceProf(section *schema.Section, profs map[schema.ProfessorKey]*schema.Professor) {
	for _, profKey := range section.Professors {
		profSectionKey := schema.ProfSectionKey{
			Subject_prefix: section.Course.Subject_prefix,
			Course_number:  section.Course.Course_number,
			Section_number: section.Section_number,
			Term:           section.Academic_session.Name,
		}

		professor, exists := profs[profKey]

		// validate if the section references some professor not in the parsed professors
		if !exists {
			log.Printf("Nonexistent professor reference found for section ID %s!", section.Id.Hex())
			log.Printf("Referenced professor key: %+v", profKey)
			log.Panic("Sections failed to validate!")
		}

		// validate if the referenced professor references back to section
		if !slices.Contains(professor.Sections, profSectionKey) {
			log.Printf("Inconsistent professor reference found for section ID %s! The section references the professor, but not vice-versa!", section.Id.Hex())
			log.Printf("Referenced professor key: %+v", profKey)
			log.Panic("Sections failed to validate!")
		}
	}
}

// Validate section reference to course
func valSectionReferenceCourse(section *schema.Section, coursesByKey map[schema.CourseKey]*schema.Course) {
	_, exists := coursesByKey[section.Course]

	// validate if section references some course not in parsed courses
	if !exists {
		log.Printf("Nonexistent course reference found for section ID %s!", section.Id.Hex())
		log.Printf("Referenced course key: %+v", section.Course)
		log.Panic("Sections failed to validate!")
	}
}

// Validate if the professors are duplicate
func valDuplicateProfs(prof1 *schema.Professor, prof2 *schema.Professor) {
	if prof1.First_name == prof2.First_name &&
		prof1.Last_name == prof2.Last_name &&
		prof1.Profile_uri == prof2.Profile_uri {
		log.Printf("Duplicate professor found!")
		log.Printf("Professor 1: %v\n\nProfessor 2: %v", prof1, prof2)
		log.Panic("Professors failed to validate!")
	}
}
