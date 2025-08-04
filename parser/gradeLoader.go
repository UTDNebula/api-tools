package parser

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func loadGrades(csvDir string) error {
	log.Print("Beginning grades loading.")

	// MAP[SEMESTER] -> MAP[SUBJECT + NUMBER + SECTION] -> GRADE DISTRIBUTION
	GradeMap = make(map[string]map[string][]int)

	dir, err := os.ReadDir(csvDir)
	if err != nil {
		return fmt.Errorf("failed to read grade directory %q: %v", csvDir, err)
	}

	for _, entry := range dir {
		if entry.IsDir() {
			continue
		}

		result, err := csvToMap(filepath.Join(csvDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to process grade file %q: %v", entry.Name(), err)
		}
		session := strings.TrimSuffix(entry.Name(), ".csv")
		GradeMap[session] = result
	}

	log.Print("Finished loading grades!")
	return nil
}

func csvToMap(csvFilePath string) (map[string][]int, error) {
	distroMap := make(map[string][]int)

	csvFile, err := os.Open(csvFilePath)
	if err != nil {
		return map[string][]int{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer csvFile.Close()

	records, err := csv.NewReader(csvFile).ReadAll() // records is [][]strings
	if err != nil {
		return map[string][]int{}, fmt.Errorf("failed to parse CSV data: %w", err)
	}

	// look for the subject column and w column
	subjectCol := -1
	catalogNumberCol := -1
	sectionCol := -1
	wCol := -1
	aPlusCol := -1

	headerRow := records[0]

	for i, header := range headerRow {
		switch header {
		case "Subject":
			subjectCol = i
		case "Catalog Number", "Catalog Nbr":
			catalogNumberCol = i
		case "Section":
			sectionCol = i
		case "W", "Total W", "W Total":
			wCol = i
		case "A+":
			aPlusCol = i
		}
		if wCol == -1 || subjectCol == -1 || catalogNumberCol == -1 || sectionCol == -1 || aPlusCol == -1 {
			continue
		} else {
			break
		}
	}

	if wCol == -1 {
		//log.Panicf("could not find W column")
	}
	if sectionCol == -1 {
		return map[string][]int{}, fmt.Errorf("could not find Section column")
	}
	if subjectCol == -1 {
		return map[string][]int{}, fmt.Errorf("could not find Subject column")
	}
	if catalogNumberCol == -1 {
		return map[string][]int{}, fmt.Errorf("could not find catalog # column")
	}
	if aPlusCol == -1 {
		return map[string][]int{}, fmt.Errorf("could not find A+ column")
	}

	for _, record := range records[1:] {
		// convert grade distribution from string to int
		intSlice := [14]int{}

		for j := 0; j < 13; j++ {
			intSlice[j], _ = strconv.Atoi(record[aPlusCol+j])
		}
		// add w number to the grade_distribution slice
		if wCol != -1 {
			intSlice[13], _ = strconv.Atoi(record[wCol])
		}

		// add new grade distribution to map, keyed by SUBJECT + NUMBER + SECTION
		// Be sure to trim left padding on section number
		trimmedSectionNumber := strings.TrimLeft(record[sectionCol], "0")
		distroKey := record[subjectCol] + record[catalogNumberCol] + trimmedSectionNumber
		distroMap[distroKey] = intSlice[:]
	}
	return distroMap, nil
}
