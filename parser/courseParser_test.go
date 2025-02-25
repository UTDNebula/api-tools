package parser

import (
	"testing"

	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestGetCourse(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			_, courseNum := getInternalClassAndCourseNum(testCase.ClassInfo)
			output := *getCourse(courseNum, testCase.Section.Academic_session, testCase.RowInfo, testCase.ClassInfo)
			expected := testCase.Course

			diff := cmp.Diff(expected, output, cmpopts.IgnoreFields(schema.Course{}, "Id"))

			if diff != "" {
				t.Errorf("Failed (-expected +got)\n %s", diff)
			}

		})
	}
}

func TestGetCatalogYear(t *testing.T) {
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
