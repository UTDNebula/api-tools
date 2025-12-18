package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/UTDNebula/nebula-api/api/schema"
)

// TestGetCourse checks course parsing from HTML fixtures.
func TestGetCourse(t *testing.T) {
	t.Parallel()

	for name, testCase := range testData {
		t.Run(name, func(t *testing.T) {
			_, courseNum := getInternalClassAndCourseNum(testCase.ClassInfo)
			output := *getCourse(courseNum, testCase.Section.Academic_session, testCase.RowInfo, testCase.ClassInfo)
			expected := testCase.Course

			diff := cmp.Diff(expected, output, cmpopts.IgnoreFields(schema.Course{}, "Id", "Enrollment_reqs", "Prerequisites", "Key", "Section_keys"))

			if diff != "" {
				t.Errorf("Failed (-expected +got)\n %s", diff)
			}

		})
	}
}

// TestGetCatalogYear ensures catalog year derivation matches expected academic sessions.
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

			defer func() {
				// Test fails if we panic when we didn't want to or didn't when we did
				if rec := recover(); rec != nil {
					if !testCase.Panic {
						t.Errorf("unexpected panic for session %q: %v", testCase.Session.Name, rec)
					}
				} else {
					if testCase.Panic {
						t.Errorf("expected panic for session %q but got none", testCase.Session.Name)
					}
				}
			}()

			// only call if we *expect* it to succeed
			output := getCatalogYear(testCase.Session)
			if !testCase.Panic && output != testCase.Expected {
				t.Errorf("expected %q, got %q", testCase.Expected, output)
			}
		})
	}
}

// TestGetPrefixAndCourseNum verifies extraction of subject prefixes and course numbers.
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
			defer func() {
				if r := recover(); r != nil {
					if !testCase.Panic {
						t.Errorf("unexpected panic for input %q: %v", name, r)
					}
				} else {
					if testCase.Panic {
						t.Errorf("expected panic for input %q but none occurred", name)
					}
				}
			}()

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
