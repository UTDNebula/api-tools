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

var grades = []string{"A+", "A", "A-", "B+", "B", "B-", "C+", "C", "C-", "D+", "D", "D-", "F", "W", "P", "CR", "NC", "I"}

func loadGrades(csvDir string) map[string]map[string][]int {

	// MAP[SEMESTER] -> MAP[SUBJECT + NUMBER + SECTION] -> GRADE DISTRIBUTION
	gradeMap := make(map[string]map[string][]int)

	if csvDir == "" {
		log.Print("No grade data CSV directory specified. Grade data will not be included.")
		return gradeMap
	}

	dirPtr, err := os.Open(csvDir)
	if err != nil {
		panic(err)
	}
	defer dirPtr.Close()

	csvFiles, err := dirPtr.ReadDir(-1)
	if err != nil {
		panic(err)
	}

	for _, csvEntry := range csvFiles {

		if csvEntry.IsDir() {
			continue
		}

		csvPath := fmt.Sprintf("%s/%s", csvDir, csvEntry.Name())

		csvFile, err := os.Open(csvPath)
		if err != nil {
			panic(err)
		}
		defer csvFile.Close()

		// Create logs directory
		if _, err := os.Stat("./logs/grades"); err != nil {
			os.Mkdir("./logs/grades", os.ModePerm)
		}

		// Create log file [name of csv].log in logs directory
		basePath := filepath.Base(csvPath)
		csvName := strings.TrimSuffix(basePath, filepath.Ext(basePath))
		logFile, err := os.Create("./logs/grades/" + csvName + ".log")

		if err != nil {
			log.Panic("Could not create CSV log file.")
		}
		defer logFile.Close()

		// Put data from csv into map
		gradeMap[csvName] = csvToMap(csvFile, logFile)
	}

	return gradeMap
}

func csvToMap(csvFile *os.File, logFile *os.File) map[string][]int {
	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll() // records is [][]strings
	if err != nil {
		log.Panicf("Error parsing %s: %s", csvFile.Name(), err.Error())
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

	// required columns
	for _, name := range []string{"Section", "Subject", "Catalog Number", "A+"} {
		if _, ok := indexMap[name]; !ok {
			fmt.Fprintf(logFile, "could not find %s column", name)
			log.Panicf("could not find %s column", name)
		}
	}

	// optional columns
	for _, name := range []string{"W", "P", "CR", "NC", "I"} {
		if _, ok := indexMap[name]; !ok {
			fmt.Fprintf(logFile, "could not find %s column\n", name)
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
	return distroMap
}
