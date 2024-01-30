// This file contains utility methods used throughout various files in this repo.

package utils

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Encodes and writes the given data as tab-indented JSON to the given filepath.
func WriteJSON(filepath string, data interface{}) error {
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

// Recursively gets the filepath of every file with the given extension, using the given directory as the root.
func GetAllFilesWithExtension(inDir string, extension string) []string {
	var filePaths []string
	err := filepath.WalkDir(inDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Add any html files (excluding evals) to sectionFilePaths
		if filepath.Ext(path) == extension {
			filePaths = append(filePaths, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return filePaths
}

// Removes standard whitespace characters (space, tab, newline, carriage return) from a given string.
func TrimWhitespace(text string) string {
	return strings.Trim(text, " \t\n\r")
}

// Gets all of the values from a given map.
func GetMapValues[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

// Gets all of the keys from a given map.
func GetMapKeys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

// Creates a regexp with MustCompile() using a sprintf input.
func Regexpf(format string, vars ...interface{}) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(format, vars...))
}
