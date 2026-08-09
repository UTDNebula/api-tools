package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

// LoadProfiles reads scraped profile API data and populates the package maps.
func LoadProfiles(inDir string) bool {
	path := fmt.Sprintf("%s/%s", inDir, profilesRawFileName)
	fptr, err := os.Open(path)
	if err != nil {
		return false
	}
	defer fptr.Close()

	payload, err := io.ReadAll(fptr)
	if err != nil {
		log.Printf("Failed to read profiles JSON: %v", err)
		return false
	}

	rows, err := decodeProfileRows(payload)
	if err != nil {
		log.Printf("Failed to decode profiles JSON: %v", err)
		return false
	}

	loadedCount := 0
	for _, row := range rows {
		if !row.Public {
			continue
		}

		prof := buildProfessorFromRow(row)
		if prof == nil {
			continue
		}

		professorKey := prof.First_name + prof.Last_name
		if existing, exists := Professors[professorKey]; exists {
			mergeProfileProfessor(existing, prof)
			continue
		}
		Professors[professorKey] = prof
		ProfessorIDMap[prof.Id] = professorKey
		loadedCount++
	}

	log.Printf("Loaded %d profiles from %s.", loadedCount, profilesRawFileName)
	return true
}

func loadProfiles(inDir string) {
	if LoadProfiles(inDir) {
		return
	}

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

// ParseProfiles loads profile data and writes only the professors output file.
func ParseProfiles(inDir string, outDir string) {
	loadProfiles(inDir)
	syncProfessorSectionLinks()

	if err := os.MkdirAll(outDir, 0777); err != nil {
		panic(err)
	}

	utils.WriteJSON(fmt.Sprintf("%s/professors.json", outDir), utils.GetMapValues(Professors))
}
