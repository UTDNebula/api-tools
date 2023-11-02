package parser

import "log"

func validate() {
	// Set up deferred handler for panics to display validation fails
	defer func() {
		if err := recover(); err != nil {
			log.Printf("VALIDATION FAILED: %s", err)
		}
	}()

	log.Printf("\nValidating courses...\n")
	courseKeys := getMapKeys(Courses)
	for i := 0; i < len(courseKeys)-1; i++ {
		course1 := Courses[courseKeys[i]]
		// Check for duplicate courses by comparing course_number, subject_prefix, and catalog_year as a compound key
		for j := i + 1; j < len(courseKeys); j++ {
			course2 := Courses[courseKeys[j]]
			if course2.Catalog_year == course1.Catalog_year && course2.Course_number == course1.Course_number && course2.Subject_prefix == course1.Subject_prefix {
				log.Printf("Duplicate course found for %s%s!\n", course1.Subject_prefix, course1.Course_number)
				log.Printf("Course 1: %v\n\nCourse 2: %v", course1, course2)
				log.Panic("Courses failed to validate!")
			}
		}
		// Make sure course isn't referencing any nonexistent sections, and that course-section references are consistent both ways
		for _, sectionId := range course1.Sections {
			section, exists := Sections[sectionId]
			if !exists {
				log.Printf("Nonexistent section reference found for %s%s!\n", course1.Subject_prefix, course1.Course_number)
				log.Printf("Referenced section ID: %s\nCourse ID: %s\n", sectionId, course1.Id)
				log.Panic("Courses failed to validate!")
			}
			if section.Course_reference != course1.Id {
				log.Printf("Inconsistent section reference found for %s%s! The course references the section, but not vice-versa!\n", course1.Subject_prefix, course1.Course_number)
				log.Printf("Referenced section ID: %s\nCourse ID: %s\nSection course reference: %s\n", sectionId, course1.Id, section.Course_reference)
				log.Panic("Courses failed to validate!")
			}
		}
	}
	courseKeys = nil
	log.Print("No invalid courses!\n\n")

	log.Print("Validating sections...\n")
	sectionKeys := getMapKeys(Sections)
	for i := 0; i < len(sectionKeys)-1; i++ {
		section1 := Sections[sectionKeys[i]]
		// Check for duplicate sections by comparing section_number, course_reference, and academic_session as a compound key
		for j := i + 1; j < len(sectionKeys); j++ {
			section2 := Sections[sectionKeys[j]]
			if section2.Section_number == section1.Section_number &&
				section2.Course_reference == section1.Course_reference &&
				section2.Academic_session == section1.Academic_session {
				log.Print("Duplicate section found!\n")
				log.Printf("Section 1: %v\n\nSection 2: %v", section1, section2)
				log.Panic("Sections failed to validate!")
			}
		}
		// Make sure section isn't referencing any nonexistent professors, and that section-professor references are consistent both ways
		for _, profId := range section1.Professors {
			professorKey, exists := ProfessorIDMap[profId]
			if !exists {
				log.Printf("Nonexistent professor reference found for section ID %s!\n", section1.Id)
				log.Printf("Referenced professor ID: %s\n", profId)
				log.Panic("Sections failed to validate!")
			}
			profRefsSection := false
			for _, profSection := range Professors[professorKey].Sections {
				if profSection == section1.Id {
					profRefsSection = true
					break
				}
			}
			if !profRefsSection {
				log.Printf("Inconsistent professor reference found for section ID %s! The section references the professor, but not vice-versa!\n", section1.Id)
				log.Printf("Referenced professor ID: %s\n", profId)
				log.Panic("Sections failed to validate!")
			}
		}
		// Make sure section isn't referencing a nonexistant course
		_, exists := CourseIDMap[section1.Course_reference]
		if !exists {
			log.Printf("Nonexistent course reference found for section ID %s!\n", section1.Id)
			log.Printf("Referenced course ID: %s\n", section1.Course_reference)
			log.Panic("Sections failed to validate!")
		}
	}
	sectionKeys = nil
	log.Printf("No invalid sections!\n\n")

	log.Printf("Validating professors...\n")
	profKeys := getMapKeys(Professors)
	// Check for duplicate professors by comparing first_name, last_name, and sections as a compound key
	for i := 0; i < len(profKeys)-1; i++ {
		prof1 := Professors[profKeys[i]]
		for j := i + 1; j < len(profKeys); j++ {
			prof2 := Professors[profKeys[j]]
			if prof2.First_name == prof1.First_name &&
				prof2.Last_name == prof1.Last_name &&
				prof2.Profile_uri == prof1.Profile_uri {
				log.Printf("Duplicate professor found!\n")
				log.Printf("Professor 1: %v\n\nProfessor 2: %v", prof1, prof2)
				log.Panic("Professors failed to validate!")
			}
		}
	}
	log.Printf("No invalid professors!\n\n")
}
