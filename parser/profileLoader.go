package parser

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

// loadProfiles loads file profiles.json from the inDir.
// If successful will update Professors and return nil, otherwise will return an error.
func loadProfiles(inDir string) error {

	profs, err := utils.UnmarshallFile[[]schema.Professor](filepath.Join(inDir, "profiles.json"))
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
