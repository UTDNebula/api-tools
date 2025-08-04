package parser

import (
	"fmt"
	"log"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

// loadProfiles loads file profiles.json from the inDir
//
//	returns nil if profiles are loaded successfully
func loadProfiles(inDir string) error {

	profs, err := utils.UnmarshallFile[[]schema.Professor](fmt.Sprintf("%s/profiles.json", inDir))
	if err != nil {
		return fmt.Errorf("failed to load profiles.json : %v", err)
	}

	log.Print("Beginning profile load.")

	for _, prof := range profs {
		professorKey := prof.First_name + prof.Last_name
		Professors[professorKey] = &prof
		ProfessorIDMap[prof.Id] = professorKey
	}

	log.Printf("Loaded %d profiles!", len(profs))
	return nil
}
