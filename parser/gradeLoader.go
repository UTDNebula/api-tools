package parser

import (
	"encoding/csv"
	"fmt"
	"github.com/UTDNebula/api-tools/utils"
	"log"
	"os"
	"strconv"
	"strings"
)

var (
	grades          = []string{"A+", "A", "A-", "B+", "B", "B-", "C+", "C", "C-", "D+", "D", "D-", "F", "W", "P", "CR", "NC", "I"}
	optionalColumns = []string{"W", "P", "CR", "NC", "I"}
	requiredColumns = []string{"Section", "Subject", "Catalog Number", "A+"}
)

func loadGrades(csvDir string) (map[string]map[string][]int, error) {
	// MAP[SEMESTER] -> MAP[SUBJECT + NUMBER + SECTION] -> GRADE DISTRIBUTION
	gradeMap := make(map[string]map[string][]int)

	fileNames := utils.GetAllFilesWithExtension(csvDir, ".csv")
	for _, name := range fileNames {

		var err error
		gradeMap[name], err = csvToMap(name)
		if err != nil {
			return gradeMap, fmt.Errorf("error parsing %s: %v", name, err)
		}
	}
	return gradeMap, nil
}

func csvToMap(filename string) (map[string][]int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening CSV file '%s': %v", filename, err)
	}

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", filename, err)
	}

	indexMap := make(map[string]int)
	for j, col := range records[0] {
		switch col {
		case "Catalog Number", "Catalog Nbr":
			indexMap["Catalog Number"] = j
		case "W", "Total W", "W Total":
			indexMap["W"] = j
		default:
			indexMap[col] = j
		}
	}

	for _, name := range requiredColumns {
		if _, ok := indexMap[name]; !ok {
			return nil, fmt.Errorf("could not find %s column in %s", name, filename)
		}
	}

	for _, name := range optionalColumns {
		if _, ok := indexMap[name]; !ok {
			log.Printf("could not find %s column in %s", name, filename)
		}
	}

	sectionCol := indexMap["Section"]
	subjectCol := indexMap["Subject"]
	catalogNumberCol := indexMap["Catalog Number"]

	distroMap := make(map[string][]int)
	for _, record := range records[1:] {
		// convert grade distribution from string to int
		intSlice := make([]int, len(grades))
		for i, col := range grades {
			if pos, ok := indexMap[col]; ok {
				intSlice[i], _ = strconv.Atoi(record[pos])
			}
		}

		// add new grade distribution to map, keyed by SUBJECT + NUMBER + SECTION
		// Be sure to trim left padding on section number
		trimmedSectionNumber := strings.TrimLeft(record[sectionCol], "0")
		distroKey := record[subjectCol] + record[catalogNumberCol] + trimmedSectionNumber
		distroMap[distroKey] = intSlice[:]
	}

	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("failed to close file '%s': %v", filename, err)
	}

	return distroMap, nil
}
