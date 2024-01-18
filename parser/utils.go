package parser

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func WriteJSON(filepath string, data interface{}) error {
	fptr, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer fptr.Close()
	encoder := json.NewEncoder(fptr)
	encoder.SetIndent("", "\t")
	encoder.Encode(GetMapValues(Courses))
	return nil
}

func GetAllSectionFilepaths(inDir string) []string {
	var sectionFilePaths []string
	err := filepath.WalkDir(inDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Add any html files (excluding evals) to sectionFilePaths
		if filepath.Ext(path) == ".html" {
			sectionFilePaths = append(sectionFilePaths, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return sectionFilePaths
}

func TrimWhitespace(text string) string {
	return strings.Trim(text, " \t\n\r")
}

func GetMapValues[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

func GetMapKeys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}
