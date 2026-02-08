package parser

import (
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func parseProfessors(sectionId schema.SectionKey, rowInfo map[string]*goquery.Selection) []schema.ProfessorKey {
	professorText := utils.TrimWhitespace(rowInfo["Instructor(s):"].Text())
	professorMatches := personRegexp.FindAllStringSubmatch(professorText, -1)
	var profRefs []schema.ProfessorKey = make([]schema.ProfessorKey, 0, len(professorMatches))
	for _, match := range professorMatches {

		nameStr := utils.TrimWhitespace(match[1])
		names := strings.Split(nameStr, " ")

		firstName := strings.Join(names[:len(names)-1], " ")
		lastName := names[len(names)-1]

		// Ignore blank names, because they exist for some reason???
		if firstName == "" || lastName == "" {
			continue
		}

		profKey := firstName + lastName

		professorKey := schema.ProfessorKey {
			First_name: firstName,
			Last_name: lastName,
		}

		prof, profExists := Professors[profKey]
		if profExists {
			prof.Sections = append(prof.Sections, sectionId)
			profRefs = append(profRefs, professorKey)
			continue
		}

		prof = &schema.Professor{}
		prof.Id = primitive.NewObjectID()
		prof.First_name = firstName
		prof.Last_name = lastName
		prof.Key = professorKey
		prof.Titles = []string{utils.TrimWhitespace(match[2])}
		prof.Email = utils.TrimWhitespace(match[3])
		prof.Sections = []schema.SectionKey{sectionId}
		profRefs = append(profRefs, professorKey)
		Professors[profKey] = prof
		ProfessorIDMap[prof.Id] = profKey
	}
	return profRefs
}
