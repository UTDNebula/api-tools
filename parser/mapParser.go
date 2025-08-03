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

// Found under "Academic & Administrative" and "Housing" on https://api.concept3d.com/categories/?map=1772&key=0001085cc708b9cef47080f064612ca5
var BUILDINGS_CATEGORY_IDS = []int{42138, 42141}

var acronymRegex = regexp.MustCompile(`.*\((.*)\)`)

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
