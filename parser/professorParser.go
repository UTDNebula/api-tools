package parser

import (
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Parse the list of professors of the section with given id.
// For each professor, set up their reference to the given section using compound key
func parseProfessors(
	sectionId primitive.ObjectID, sectionNumber string, rowInfo map[string]*goquery.Selection,
	course schema.Course, session schema.AcademicSession,
) []string {

	professorText := utils.TrimWhitespace(rowInfo["Instructor(s):"].Text())
	professorMatches := personRegexp.FindAllStringSubmatch(professorText, -1)
	profRefs := make([]string, 0, len(professorMatches))

	// Each professor's reference to the given section
	profSectionRef := schema.ProfSectionRef{
		Prefix:         course.Subject_prefix,
		Number:         course.Course_number,
		Term:           session.Name,
		Section_number: sectionNumber,
	}

	for _, match := range professorMatches {
		nameStr := utils.TrimWhitespace(match[1])
		names := strings.Split(nameStr, " ")

		firstName := strings.Join(names[:len(names)-1], " ")
		lastName := names[len(names)-1]

		// Ignore blank names, because they exist for some reason???
		if firstName == "" || lastName == "" {
			continue
		}

		prof, profExists := Professors[firstName+" "+lastName]
		if profExists {
			prof.Sections = append(prof.Sections, &profSectionRef)
			// Under the assumption that existent prof has email recorded
			profRefs = append(profRefs, firstName+" "+lastName)
			continue
		}

		prof = &schema.Professor{
			Id:         primitive.NewObjectID(),
			First_name: firstName,
			Last_name:  lastName,
			Titles:     []string{utils.TrimWhitespace(match[2])},
			Email:      utils.TrimWhitespace(match[3]),
			Sections:   []*schema.ProfSectionRef{&profSectionRef},
		}
		profRefs = append(profRefs, prof.First_name+" "+prof.Last_name)
		Professors[firstName+" "+lastName] = prof
	}
	return profRefs
}
