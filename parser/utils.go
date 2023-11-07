package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func writeJSON(filepath string, data interface{}) error {
	fptr, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer fptr.Close()
	encoder := json.NewEncoder(fptr)
	encoder.SetIndent("", "\t")
	encoder.Encode(data)
	return nil
}

// TODO: Do this in a cleaner manner via filepath.Walk or similar
func getAllSectionFilepaths(inDir string) []string {
	var sectionFilePaths []string
	// Try to open inDir
	fptr, err := os.Open(inDir)
	if err != nil {
		panic(err)
	}
	// Try to get term directories in inDir
	termFiles, err := fptr.ReadDir(-1)
	fptr.Close()
	if err != nil {
		panic(err)
	}
	// Iterate over term directories
	for _, file := range termFiles {
		if !file.IsDir() {
			continue
		}
		termPath := fmt.Sprintf("%s/%s", inDir, file.Name())
		fptr, err = os.Open(termPath)
		if err != nil {
			panic(err)
		}
		courseFiles, err := fptr.ReadDir(-1)
		fptr.Close()
		if err != nil {
			panic(err)
		}
		// Iterate over course directories
		for _, file := range courseFiles {
			coursePath := fmt.Sprintf("%s/%s", termPath, file.Name())
			fptr, err = os.Open(coursePath)
			if err != nil {
				panic(err)
			}
			sectionFiles, err := fptr.ReadDir(-1)
			fptr.Close()
			if err != nil {
				panic(err)
			}
			// Get all section file paths from course directory
			for _, file := range sectionFiles {
				sectionFilePaths = append(sectionFilePaths, fmt.Sprintf("%s/%s", coursePath, file.Name()))
			}
		}
	}
	return sectionFilePaths
}

func trimWhitespace(text string) string {
	return strings.Trim(text, " \t\n\r")
}

func getMapValues[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

func getMapKeys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}
