package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/nebula-api/api/schema"
)

type TestData struct {
	RowInfo   map[string]*goquery.Selection
	ClassInfo map[string]string
	Section   schema.Section
	Course    schema.Course
}

var testDataCache map[string]TestData

func loadTestData(t *testing.T) {
	t.Helper()
	if testDataCache != nil {
		return
	}

	testDataCache = make(map[string]TestData)
	dir, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("Failed to load testdata: %v", err)
	}

	for _, file := range dir {
		if !file.IsDir() {
			continue
		}
		testCase, err := loadTest(file.Name())
		if err != nil {
			t.Fatalf("Failed to load %s: %v", file.Name(), err)
		}
		testDataCache[file.Name()] = testCase
	}
}

func loadTest(dir string) (TestData, error) {

	htmlBytes, err := os.ReadFile(fmt.Sprintf("testdata/%s/input.html", dir))
	if err != nil {
		return TestData{}, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBytes))
	if err != nil {
		return TestData{}, err
	}

	result := TestData{
		RowInfo:   getRowInfo(doc),
		ClassInfo: getClassInfo(doc),
	}

	jsonBytes, err := os.ReadFile(fmt.Sprintf("testdata/%s/section.json", dir))
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(jsonBytes, &result.Section)
	if err != nil {
		return TestData{}, err
	}

	jsonBytes, err = os.ReadFile(fmt.Sprintf("testdata/%s/course.json", dir))
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(jsonBytes, &result.Course)
	if err != nil {
		return TestData{}, err
	}

	return result, nil
}
