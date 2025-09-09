package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/UTDNebula/nebula-api/api/schema"
)

func TestGetCourse(t *testing.T) {
	t.Parallel()

	for name, testCase := range testData {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, courseNum := getInternalClassAndCourseNum(testCase.ClassInfo)
			output := *getCourse(courseNum, testCase.Section.Academic_session, testCase.RowInfo, testCase.ClassInfo)
			expected := testCase.Course

			// skip fields that use primitive.ObjectID or are populated by ReqParser they are already
			// covered in parser_test and are awkward to implement here, also mostly out of the scope of course parser
			diff := cmp.Diff(expected, output, cmpopts.IgnoreFields(schema.Course{}, "Id", "Sections", "Enrollment_reqs", "Prerequisites"))

			if diff != "" {
				t.Errorf("Failed (-expected +got)\n %s", diff)
			}

		})
	}
}

func TestGetCatalogYear(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		Session  schema.AcademicSession
		Expected string
		Panic    bool
	}{
		"Case_001": {
			Session:  schema.AcademicSession{Name: "25S"},
			Expected: "24",
		},
		"Case_002": {
			Session:  schema.AcademicSession{Name: "25F"},
			Expected: "25",
		},
		"Case_003": {
			Session:  schema.AcademicSession{Name: "22U"},
			Expected: "21",
		},
		"Case_004": {
			Session:  schema.AcademicSession{Name: "20S"},
			Expected: "19",
		},
		"Case_005": {
			Session: schema.AcademicSession{Name: "Garbage"},
			Panic:   true,
		},
		"Case_006": {
			Session: schema.AcademicSession{Name: "20P"},
			Panic:   true,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.Panic {
				defer FailTestIfNoPanic(t, name)
			} else {
				defer FailTestIfPanic(t, name)
			}

			output := getCatalogYear(testCase.Session)
			if !testCase.Panic && output != testCase.Expected {
				t.Errorf("expected %q, got %q", testCase.Expected, output)
			}
		})
	}
}

func TestGetPrefixAndCourseNum(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		ClassInfo map[string]string
		Prefix    string
		Number    string
		Panic     bool
	}{
		"Case_001": {
			ClassInfo: map[string]string{
				"Class Section:": "ACCT2301.001.25S",
			},
			Prefix: "ACCT",
			Number: "2301",
		},
		"Case_002": {
			ClassInfo: map[string]string{
				"Class Section:": "ENTP3301.002.24S",
			},
			Prefix: "ENTP",
			Number: "3301",
		},
		"Case_003": {
			ClassInfo: map[string]string{
				"Class Section:": "Garbage In, Garbage out",
			},
			Panic: true,
		},
		"Case_004": {
			ClassInfo: map[string]string{
				"Class Section:": "ENTP33S",
			},
			Panic: true,
		},
		"Case_005": {
			ClassInfo: map[string]string{
				"Class Section:": "",
			},
			Panic: true,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.Panic {
				defer FailTestIfNoPanic(t, name)
			} else {
				defer FailTestIfPanic(t, name)
			}

			prefix, number := getPrefixAndNumber(testCase.ClassInfo)

			if !testCase.Panic {
				if prefix != testCase.Prefix {
					t.Errorf("expected %q got %q", testCase.Prefix, prefix)
				}
				if number != testCase.Number {
					t.Errorf("expected %q got %q", testCase.Number, number)
				}
			}
		})
	}
}
