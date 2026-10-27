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

// rawBrochure is the part of a Terradotta brochure that the parser keeps.
type rawBrochure struct {
	ProgramID int `json:"programId"`
	Current   struct {
		Sections []rawSection `json:"sections"`
	} `json:"current"`
	NextCost *rawCost `json:"nextAppCycleCostSheet"`
}

type rawSection struct {
	DisplayName string      `json:"sectionDisplayName"`
	Widgets     []rawWidget `json:"sectionWidgets"`
}

type rawWidget struct {
	ContentType string `json:"contentType"`
	ContentHTML string `json:"contentHTML"`
	// "" for most widgets, an object holding the parameters for "information sheet" widgets
	InformationSheet json.RawMessage `json:"contentInformationSheet"`
}

type rawCost struct {
	Term        string        `json:"term"`
	Year        int           `json:"year"`
	Billable    []rawCostItem `json:"billableCostSheetItems"`
	NonBillable []rawCostItem `json:"nonBillableCostSheetItems"`
	Credits     []rawCostItem `json:"creditCostSheetItems"`
}

type rawCostItem struct {
	Name  string `json:"costSheetItemName"`
	Type  string `json:"costSheetItemType"`
	Costs []struct {
		Key   string  `json:"costKey"`
		Value float64 `json:"costValue"`
	} `json:"costs"`
}

func parseBrochure(path string) (schema.Program, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return schema.Program{}, err
	}

	var brochure rawBrochure
	if err := json.Unmarshal(content, &brochure); err != nil {
		return schema.Program{}, err
	}

	program := schema.Program{
		ID:       brochure.ProgramID,
		Sections: make([]schema.ProgramSection, 0, len(brochure.Current.Sections)),
	}
	for _, rawSection := range brochure.Current.Sections {
		section := schema.ProgramSection{
			DisplayName: rawSection.DisplayName,
			Widgets:     make([]schema.ProgramWidget, 0, len(rawSection.Widgets)),
		}
		for _, rawWidget := range rawSection.Widgets {
			parameters, err := parseInformationSheet(rawWidget.InformationSheet)
			if err != nil {
				return schema.Program{}, fmt.Errorf("section %q: %w", rawSection.DisplayName, err)
			}
			section.Widgets = append(section.Widgets, schema.ProgramWidget{
				ContentType: rawWidget.ContentType,
				ContentHTML: rawWidget.ContentHTML,
				Parameters:  parameters,
			})
		}
		program.Sections = append(program.Sections, section)
	}

	if brochure.NextCost != nil {
		program.NextCost = &schema.Cost{
			Term:        brochure.NextCost.Term,
			Year:        brochure.NextCost.Year,
			Billable:    convertCostItems(brochure.NextCost.Billable),
			NonBillable: convertCostItems(brochure.NextCost.NonBillable),
			Credits:     convertCostItems(brochure.NextCost.Credits),
		}
	}
	return program, nil
}

// parseInformationSheet returns the parameters of an information sheet, or nil when the widget has none.
func parseInformationSheet(raw json.RawMessage) ([]schema.Parameter, error) {
	if len(raw) == 0 || raw[0] != '{' {
		return nil, nil
	}
	var sheet struct {
		Parameters []struct {
			Name           string `json:"parameterName"`
			AssignedValues []any  `json:"assignedValues"` // strings, except YESNO parameters which hold bools
		} `json:"parameters"`
	}
	if err := json.Unmarshal(raw, &sheet); err != nil {
		return nil, fmt.Errorf("parse information sheet: %w", err)
	}

	parameters := make([]schema.Parameter, 0, len(sheet.Parameters))
	for _, rawParameter := range sheet.Parameters {
		values := make([]string, 0, len(rawParameter.AssignedValues))
		for _, value := range rawParameter.AssignedValues {
			switch value := value.(type) {
			case string:
				values = append(values, value)
			case bool:
				if value {
					values = append(values, "Yes")
				} else {
					values = append(values, "No")
				}
			default:
				return nil, fmt.Errorf("parameter %q: unexpected value %v", rawParameter.Name, value)
			}
		}
		parameters = append(parameters, schema.Parameter{Name: rawParameter.Name, AssignedValues: values})
	}
	return parameters, nil
}

// convertCostItems turns each item's list of {costKey, costValue} pairs into a map from key to value.
func convertCostItems(rawItems []rawCostItem) []schema.CostItem {
	items := make([]schema.CostItem, 0, len(rawItems))
	for _, rawItem := range rawItems {
		costs := make(map[string]float64, len(rawItem.Costs))
		for _, cost := range rawItem.Costs {
			costs[cost.Key] = cost.Value
		}
		items = append(items, schema.CostItem{Name: rawItem.Name, Type: rawItem.Type, Costs: costs})
	}
	return items
}
