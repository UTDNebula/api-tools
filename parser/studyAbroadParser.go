/*
	This file contains the code for the study abroad program parser.
*/

package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/rest/schema"
)

// discoveryRecord holds the program metadata that only the discovery search returns.
type discoveryRecord struct {
	ProgramID   int                 `json:"Program_Id"`
	ProgramName string              `json:"Program_Name"`
	Summary     string              `json:"Summary"`
	Terms       []string            `json:"Terms"`
	Locations   []discoveryLocation `json:"stLocations"`
}

type discoveryLocation struct {
	Country   string  `json:"country"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// ParseStudyAbroad combines the scraped program brochures with their discovery search records
// and writes the resulting programs to {outDir}/studyAbroad.json.
func ParseStudyAbroad(inDir string, outDir string) {
	studyAbroadDir := filepath.Join(inDir, "studyAbroad")

	records, err := loadDiscoveryRecords(filepath.Join(studyAbroadDir, "discoveryScraped.json"))
	if err != nil {
		log.Panic(err)
	}

	brochurePaths := utils.GetAllFilesWithExtension(filepath.Join(studyAbroadDir, "brochures"), ".json")
	if len(brochurePaths) == 0 {
		log.Panicf("No brochures found in %s; run -scrape -studyAbroad first", studyAbroadDir)
	}

	programs := make([]schema.Program, 0, len(brochurePaths))
	missingRecords := 0

	for _, path := range brochurePaths {
		program, err := parseBrochure(path)
		if err != nil {
			log.Printf("parse brochure %s: %v", path, err)
			continue
		}

		record, ok := records[program.ID]
		if !ok {
			missingRecords++
		}
		program.Name = record.ProgramName
		program.Summary = record.Summary
		program.Terms = record.Terms
		program.Locations = make([]schema.ProgramLocation, 0, len(record.Locations))
		for _, location := range record.Locations {
			program.Locations = append(program.Locations, schema.ProgramLocation{
				City:      location.City,
				Country:   location.Country,
				Latitude:  location.Latitude,
				Longitude: location.Longitude,
			})
		}

		programs = append(programs, program)
	}

	sort.Slice(programs, func(i, j int) bool { return programs[i].ID < programs[j].ID })

	if missingRecords > 0 {
		log.Printf("%d programs had no discovery record, so they have no name, terms, or locations", missingRecords)
	}
	log.Printf("Parsed %d/%d study abroad programs!", len(programs), len(brochurePaths))

	if err := utils.WriteJSON(fmt.Sprintf("%s/studyAbroad.json", outDir), programs); err != nil {
		log.Panic(err)
	}
}

// loadDiscoveryRecords reads the discovery search records keyed by program ID.
// A missing file isn't an error, since scraping with PROGRAM_IDS skips discovery.
func loadDiscoveryRecords(path string) (map[int]discoveryRecord, error) {
	records := make(map[int]discoveryRecord)

	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		log.Printf("No discovery records at %s", path)
		return records, nil
	}
	if err != nil {
		return nil, err
	}

	var list []discoveryRecord
	if err := json.Unmarshal(content, &list); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	for _, record := range list {
		records[record.ProgramID] = record
	}
	return records, nil
}

func parseBrochure(path string) (schema.Program, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return schema.Program{}, err
	}

	var program schema.Program
	if err := json.Unmarshal(content, &program); err != nil {
		return schema.Program{}, err
	}
	return program, nil
}
