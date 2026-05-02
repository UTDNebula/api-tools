package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	gradeLoaderTestCases = map[string]struct {
		csvContent string
		want       map[string][]int
		fail       bool
	}{
		"Valid_Data": {
			csvContent: `Instructor 1,Instructor 2,Instructor 3,Instructor 4,Instructor 5,Instructor 6,Subject,"Catalog Nbr",Section,A+,A,A-,B+,B,B-,C+,C,C-,D+,D,D-,F,NF,CR,I,NC,P,W
"Curchack, Fred",,,,,,AP,3300,501,6,4,2,2,1,3,1,1,,,,,1,,,,,,0
"Anjum, Zafar",,,,,,ARAB,1311,001,,26,,,1,,,,,,,,,,,,,,2`,
			want: map[string][]int{
				"AP3300501": {6, 4, 2, 2, 1, 3, 1, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0},
				"ARAB13111": {0, 26, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0},
			},
			fail: false,
		},
		"Missing_Required_Column_A+": {
			csvContent: `Subject,"Catalog Nbr",Section,A,A-,B+
CS,1337,001,10,5,5`,
			fail: true,
		},
		"Missing_Required_Column_Subject": {
			csvContent: `Instructor,"Catalog Nbr",Section,A+,A
Doe,1337,001,10,5`,
			fail: true,
		},
		"Empty_File": {
			csvContent: ``,
			fail:       true,
		},
	}
)

func TestLoadGrades(t *testing.T) {

	invalidCSVNames := []string{"22", "2F", "2022F", "20-U", "15Fall"}

	for i, name := range invalidCSVNames {
		t.Run(
			fmt.Sprintf("Invalid_CSV_Name_%d", i), func(t *testing.T) {
				tempDir := t.TempDir()

				temp, err := os.Create(filepath.Join(tempDir, name+".csv"))
				if err != nil {
					t.Errorf("failed to create temp file: %v", err)
				}
				defer temp.Close()

				_, err = loadGrades(tempDir)
				if err == nil {
					t.Errorf("expected error but got none")
				}
			},
		)
	}

	validCSVNames := []string{"25F", "18U", "26S"}
	for i, name := range validCSVNames {
		t.Run(
			fmt.Sprintf("Valid_CSV_Name_%d", i), func(t *testing.T) {
				tempDir := t.TempDir()

				temp, err := os.Create(filepath.Join(tempDir, name+".csv"))
				if err != nil {
					t.Errorf("failed to create temp file: %v", err)
				}
				defer temp.Close()

				_, err = temp.WriteString(gradeLoaderTestCases["Valid_Data"].csvContent)
				if err != nil {
					t.Errorf("failed to write test data: %v", err)
				}

				_, err = loadGrades(tempDir)
				if err != nil {
					t.Errorf("valid .csv failed: %v", err)
				}
			},
		)
	}

	t.Run("Real_Data", func(t *testing.T) {
		_, err := loadGrades("../static-data/grades/")
		if err != nil {
			t.Errorf("failed to load grades: %v", err)
		}
	})
}

func TestCSVToMap(t *testing.T) {
	tempDir := t.TempDir()

	for name, testCase := range gradeLoaderTestCases {
		t.Run(name, func(t *testing.T) {

			temp, err := os.CreateTemp(tempDir, "grades*.csv")
			if err != nil {
				t.Errorf("failed to create temp file: %v", err)
			}
			defer temp.Close()

			if _, err = temp.WriteString(testCase.csvContent); err != nil {
				t.Errorf("failed to write test data: %v", err)
			}

			output, err := csvToMap(temp.Name())
			if err != nil {
				if testCase.fail {
					return
				}
				t.Errorf("failed to load csv: %v", err)
			} else if testCase.fail {
				t.Errorf("expected failure but got none")
			} else {
				diff := cmp.Diff(testCase.want, output)
				if diff != "" {
					t.Errorf("Failed (-expected +got)\n %s", diff)
				}
			}

		})
	}
}
