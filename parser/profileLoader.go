package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/UTDNebula/nebula-api/api/schema"
)

func loadProfiles(inDir string) {
	fptr, err := os.Open(fmt.Sprintf("%s/profiles.json", inDir))
	if err != nil {
		log.Print("Couldn't find/open profiles.json in the input directory. Skipping profile load.")
		return
	}

	decoder := json.NewDecoder(fptr)

	log.Print("Beginning profile load.")

	// Read open bracket
	_, err = decoder.Token()
	if err != nil {
		panic(err)
	}

	// While the array contains values
	profileCount := 0
	for ; decoder.More(); profileCount++ {
		// Decode a professor
		var prof schema.Professor
		err := decoder.Decode(&prof)
		if err != nil {
			panic(err)
		}
		professorKey := prof.First_name + prof.Last_name
		Professors[professorKey] = &prof
		ProfessorIDMap[prof.Id] = professorKey
	}

	// Read closing bracket
	_, err = decoder.Token()
	if err != nil {
		panic(err)
	}

	log.Printf("Loaded %d profiles!", profileCount)
	fptr.Close()
}
