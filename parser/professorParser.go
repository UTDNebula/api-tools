package parser

import (
	"strings"

	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func parseProfessors(sectionId primitive.ObjectID, rowInfo map[string]string, classInfo map[string]string) []primitive.ObjectID {
	professorText := rowInfo["Instructor(s):"]
	professorMatches := personRegexp.FindAllStringSubmatch(professorText, -1)
	var profRefs []primitive.ObjectID = make([]primitive.ObjectID, 0, len(professorMatches))
	for _, match := range professorMatches {

		nameStr := match[1]
		names := strings.Split(nameStr, " ")

		firstName := names[0]
		lastName := names[len(names)-1]

		profKey := firstName + lastName

		prof, profExists := Professors[profKey]
		if profExists {
			prof.Sections = append(prof.Sections, sectionId)
			profRefs = append(profRefs, prof.Id)
			continue
		}

		prof = &schema.Professor{}
		prof.Id = primitive.NewObjectID()
		prof.First_name = firstName
		prof.Last_name = lastName
		prof.Titles = []string{match[2]}
		prof.Email = match[3]
		prof.Sections = []primitive.ObjectID{sectionId}
		profRefs = append(profRefs, prof.Id)
		Professors[profKey] = prof
		ProfessorIDMap[prof.Id] = profKey
	}
	return profRefs
}
