package parser

import (
	"testing"

	"github.com/UTDNebula/nebula-api/api/schema"
)

func TestGetCatalogYear(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		Session  schema.AcademicSession
		Expected string
	}{
		"Case_001": {
			Session: schema.AcademicSession{
				Name: "25S",
			},
			Expected: "24",
		}, "Case_002": {
			Session: schema.AcademicSession{
				Name: "25F",
			},
			Expected: "25",
		}, "Case_003": {
			Session: schema.AcademicSession{
				Name: "22U",
			},
			Expected: "21",
		}, "Case_004": {
			Session: schema.AcademicSession{
				Name: "20S",
			},
			Expected: "19",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			output := getCatalogYear(tc.Session)

			if output != tc.Expected {
				t.Errorf("expected %s got %s", tc.Expected, output)
			}
		})

	}
}

func TestGetPrefixAndCourseNum(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		classInfo map[string]string
		prefix    string
		number    string
	}{
		"Case_001": {
			classInfo: map[string]string{
				"Class Section:": "ACCT2301.001.25S",
			},
			prefix: "ACCT",
			number: "2301",
		},
		"Case_002": {
			classInfo: map[string]string{
				"Class Section:": "ENTP3301.002.24S",
			},
			prefix: "ENTP",
			number: "3301",
		},
		"Case_003": {
			classInfo: map[string]string{
				"Class Section:": "Garbage In, Garbage out",
			},
			prefix: "",
			number: "",
		},
		"Case_004": {
			classInfo: map[string]string{
				"Class Section:": "ENTP33S",
			},
			prefix: "",
			number: "",
		},
		"Case_005": {
			classInfo: map[string]string{
				"Class Section:": "",
			},
			prefix: "",
			number: "",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			prefix, number := getPrefixAndNumber(testCase.classInfo)

			if prefix != testCase.prefix {
				t.Errorf("expected %s got %s", testCase.prefix, prefix)
			}
			if number != testCase.number {
				t.Errorf("expected %s got %s", testCase.number, number)
			}
		})
	}
}
