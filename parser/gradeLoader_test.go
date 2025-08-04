package parser

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var csvHeader = []string{
	"Instructor 1", "Instructor 2", "Instructor 3", "Instructor 4", "Instructor 5", "Instructor 6", "Subject",
	"Catalog Nbr", "Section", "A+", "A", "A-", "B+", "B", "B-", "C+", "C", "C-", "D+", "D", "D-", "F", "NF",
	"CR", "I", "NC", "P", "W",
}

func TestCsvToMap(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	cases := map[string]struct {
		subject       string
		catalogNumber string
		sectionNumber string
		key           string
		gradesInput   []string
		grades        []int
	}{
		"case_001": {
			subject:       "ECS",
			catalogNumber: "2301",
			sectionNumber: "001",
			key:           "ECS23011",
			gradesInput:   []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13"},
			grades:        []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
		},
		"case_002": {
			subject:       "CS",
			catalogNumber: "1337",
			sectionNumber: "001",
			key:           "CS13371", // Test empty grade fields
			gradesInput:   []string{"", "", "", "", "", "", "", "", "", "", "", "", "", ""},
			grades:        []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		"case_003": {
			subject:       "MTHE",
			catalogNumber: "5V06",
			sectionNumber: "5H2", // Test non integer section
			key:           "MTHE5V065H2",
			gradesInput:   []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13"},
			grades:        []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fileName := filepath.Join(tempDir, fmt.Sprintf("%s.csv", name))

			if err := buildCsv(t, fileName, testCase.subject, testCase.catalogNumber, testCase.sectionNumber, testCase.gradesInput); err != nil {
				t.Errorf("failed to build csv file %v", err)
			}

			output, err := csvToMap(fileName)
			if err != nil {
				t.Errorf("failed to convert CSV to map: %v", err)
			}

			if result, ok := output[testCase.key]; !ok {
				t.Errorf("failed to find key %s in output", testCase.key)
			} else {
				// check to see that the output contains the correct data for grades
				// 0-13 is the best test since it helps show if the offset is wrong
				diff := cmp.Diff(result, testCase.grades)

				if diff != "" {
					t.Errorf("result mismatch (-want +got):\n%s", diff)
				}

			}

		})
	}

}

func buildCsv(t *testing.T, fileName string, subject string, catalogNumber string, sectionNumber string, gradesInput []string) error {
	t.Helper()
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("cannot create test file %s: %v", fileName, err)
	}
	writer := csv.NewWriter(file)

	if err = writer.Write(csvHeader); err != nil {
		return fmt.Errorf("cannot write to file %s : %v", fileName, err)
	}

	line := make([]string, 0, 28)
	line = append(line, "", "", "", "", "", "", subject, catalogNumber, sectionNumber)
	line = append(line, gradesInput[0:13]...)
	line = append(line, "", "", "", "", "", gradesInput[13])

	if err = writer.Write(line); err != nil {
		return fmt.Errorf("cannot write to file %s : %v", fileName, err)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("error flushing writer: %w", err)
	}
	return file.Close()
}
