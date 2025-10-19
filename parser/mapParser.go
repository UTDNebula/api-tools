package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

// BUILDINGS_CATEGORY_IDS lists category identifiers for academic, administrative, and housing buildings on Concept3D.
var BUILDINGS_CATEGORY_IDS = []int{42138, 42141}

var acronymRegex = regexp.MustCompile(`.*\((.*)\)`)

// ParseMapLocations filters Concept3D location exports to building records and writes normalized JSON output.
func ParseMapLocations(inDir string, outDir string) {
	mapFile, err := os.ReadFile(inDir + "/mapLocationsScraped.json")
	if err != nil {
		panic(err)
	}

	var rawData []map[string]interface{}
	err = json.Unmarshal(mapFile, &rawData)
	if err != nil {
		panic(err)
	}

	var filtered []schema.MapBuilding

	for _, rawPlace := range rawData {
		categoryPtr := utils.ConvertFromInterface[float64](rawPlace["catId"])
		if categoryPtr == nil {
			continue
		}
		if !slices.Contains(BUILDINGS_CATEGORY_IDS, int(*categoryPtr)) {
			continue
		}

		namePtr := utils.ConvertFromInterface[string](rawPlace["name"])
		if namePtr == nil {
			continue
		}
		acronym := acronymRegex.FindStringSubmatch(*namePtr)
		var acronymPtr *string = nil
		if len(acronym) > 1 {
			acronymPtr = &acronym[1]
		}

		filtered = append(filtered, schema.MapBuilding{
			Name:    namePtr,
			Acronym: acronymPtr,
			Lat:     utils.ConvertFromInterface[float64](rawPlace["lat"]),
			Lng:     utils.ConvertFromInterface[float64](rawPlace["lng"]),
		})
	}

	log.Print("Parsed Map Locations!")

	utils.WriteJSON(fmt.Sprintf("%s/mapLocations.json", outDir), filtered)
}
