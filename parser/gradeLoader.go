package parser

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/UTDNebula/api-tools/utils"
)

var (
	grades          = []string{"A+", "A", "A-", "B+", "B", "B-", "C+", "C", "C-", "D+", "D", "D-", "F", "W", "P", "CR", "NC", "I"}
	optionalColumns = []string{"W", "P", "CR", "NC", "I"}
	requiredColumns = []string{"Section", "Subject", "Catalog Number", "A+"}
	semesterRegex   = regexp.MustCompile(`[1-9][0-9][USF]`)
)

func loadGrades(csvDir string) (map[string]map[string][]int, error) {
	// MAP[SEMESTER] -> MAP[SUBJECT + NUMBER + SECTION] -> GRADE DISTRIBUTION
	gradeMap := make(map[string]map[string][]int)

	fileNames := utils.GetAllFilesWithExtension(csvDir, ".csv")
	for _, name := range fileNames {

		semester := semesterRegex.FindString(name)
		if semester == "" {
			return gradeMap, fmt.Errorf("invalid name %s, must match format {>10}{F,S,U} i.e. 22F", name)
		}

		var err error
		gradeMap[semester], err = csvToMap(name)
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
	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			log.Printf("failed to close file '%s': %v", filename, err)
		}
	}(file)

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", filename, err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV file '%s'", filename)
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

	return distroMap, nil
}
