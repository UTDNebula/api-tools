package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/nebula-api/rest/schema"
	"github.com/google/go-cmp/cmp"
)

func TestGetEnrollment(t *testing.T) {
	fixtures, err := unmarshallFile[[]struct{ Name, HTML string }]("fixtures/enrollment.json")
	if err != nil {
		t.Fatal(err)
	}
	scrapedAt := time.Date(2026, 8, 24, 12, 34, 56, 0, time.UTC)
	expected := map[string]*schema.Enrollment{
		"closed":        {Scraped_at: &scrapedAt, Status: "CLOSED/FULL", Available_seats: new(0), Enrolled_total: new(68), Waitlist: new(0)},
		"cross_listed":  {Scraped_at: &scrapedAt, Status: "OPEN", Available_seats: new(10), Enrolled_total: new(14), Waitlist: new(0)},
		"over_enrolled": {Scraped_at: &scrapedAt, Status: "CLOSED/FULL", Available_seats: new(-15), Enrolled_total: new(57), Waitlist: new(0)},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(fixture.HTML))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expected[fixture.Name], getEnrollment(getRowInfo(doc), &scrapedAt)); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
	if got := getEnrollment(nil, &scrapedAt); got != nil {
		t.Errorf("missing enrollment row = %v, want nil", got)
	}
}

func TestGetScrapedAt(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "26f", "cp_cs", "section.html")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if got := getScrapedAt(path); got != nil {
		t.Fatalf("missing timestamp file = %v, want nil", got)
	}
	if err := os.WriteFile(filepath.Join(root, "26f", "scraped_at.txt"), []byte("2026-08-24T12:34:56Z\n"), 0644); err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 24, 12, 34, 56, 0, time.UTC)
	if got := getScrapedAt(path); got == nil || !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("scraped_at = %v, want UTC %v", got, want)
	}
}

// TestGetInternalClassAndCourseNum checks parsing of internal course identifiers.
func TestGetInternalClassAndCourseNum(t *testing.T) {
	t.Parallel()

	for name, testCase := range testData {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			classNum, courseNum := getInternalClassAndCourseNum(testCase.ClassInfo)
			expectedClassNum := testCase.Section.Internal_class_number
			expectedCourseNumber := testCase.Course.Internal_course_number

			if classNum != expectedClassNum {
				t.Errorf("Class Number: expected %s got %s", expectedClassNum, classNum)
			}

			if courseNum != expectedCourseNumber {
				t.Errorf("Class Number: expected %s got %s", expectedCourseNumber, courseNum)
			}

		})
	}

	fails := []map[string]string{
		{
			"Class Section:": "ENTP33S",
		},
		{
			"Class Section:": "",
		},
		{
			"Class Section:": "Garbage In, Garbage out",
		},
	}

	for i, fail := range fails {
		name := fmt.Sprintf("case_%03d", i+len(testData))

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for input %s but none occurred", fail)
				}
			}()
			getInternalClassAndCourseNum(fail)

		})
	}
}

// TestGetAcademicSession ensures term metadata is parsed correctly.
func TestGetAcademicSession(t *testing.T) {
	t.Parallel()

	for name, testCase := range testData {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			output := getAcademicSession(testCase.RowInfo)
			expected := testCase.Section.Academic_session

			diff := cmp.Diff(expected, output)

			if diff != "" {
				t.Errorf("Failed (-expected +got)\n %s", diff)
			}
		})
	}
}

// TestGetSectionNumber validates extraction of section numbers.
func TestGetSectionNumber(t *testing.T) {
	t.Parallel()

	for name, testCase := range testData {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			output := getSectionNumber(testCase.ClassInfo)
			expected := testCase.Section.Section_number

			if output != expected {
				t.Errorf("expected %s got %s", expected, output)
			}
		})
	}

	fails := []map[string]string{
		{
			"Class Section:": "ENTP33S",
		},
		{
			"Class Section:": "",
		},
		{
			"Class Section:": "Garbage In, Garbage out",
		},
	}

	for i, fail := range fails {
		name := fmt.Sprintf("case_%03d", i+len(testData))

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for input %s but none occurred", fail)
				}
			}()
			getSectionNumber(fail)

		})

	}
}

func TestGetTeachingAssistants(t *testing.T) {
	t.Parallel()

	for name, testCase := range testData {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			output := getTeachingAssistants(testCase.RowInfo)
			expected := testCase.Section.Teaching_assistants

			diff := cmp.Diff(expected, output)

			if diff != "" {
				t.Errorf("Failed (-expected +got)\n %s", diff)
			}
		})
	}
}

func TestGetInstructionMode(t *testing.T) {
	t.Parallel()

	for name, testCase := range testData {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			output := getInstructionMode(testCase.ClassInfo)
			expected := testCase.Section.Instruction_mode

			if output != expected {
				t.Errorf("expected %s got %s", expected, output)
			}
		})

	}
}

func TestGetMeetings(t *testing.T) {
	t.Parallel()

	for name, testCase := range testData {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			output := getMeetings(testCase.RowInfo)
			expected := testCase.Section.Meetings

			diff := cmp.Diff(expected, output)

			if diff != "" {
				t.Errorf("Failed (-expected +got)\n %s", diff)
			}
		})
	}
}

func TestGetCoreFlags(t *testing.T) {
	t.Parallel()

	for name, testCase := range testData {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			output := getCoreFlags(testCase.RowInfo)
			expected := testCase.Section.Core_flags

			diff := cmp.Diff(expected, output)

			if diff != "" {
				t.Errorf("Failed (-expected +got)\n %s", diff)
			}
		})
	}
}

func TestGetSyllabusUri(t *testing.T) {
	t.Parallel()

	for name, testCase := range testData {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			output := getSyllabusUri(testCase.RowInfo)
			expected := testCase.Section.Syllabus_uri

			if output != expected {
				t.Errorf("expected %s got %s", expected, output)
			}
		})

	}
}

func TestParseTimeOrPanic(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		Input    string
		Expected time.Time
		Panic    bool
	}{
		"Case_001": {
			Input:    "January 2, 2006",
			Expected: time.Date(2006, time.January, 2, 0, 0, 0, 0, timeLocation),
			Panic:    false,
		},
		"Case_002": {
			Input:    "March 15, 2020",
			Expected: time.Date(2020, time.March, 15, 0, 0, 0, 0, timeLocation),
			Panic:    false,
		},
		"Case_003": {
			Input: "15 March, 2020", // wrong format
			Panic: true,
		},
		"Case_004": {
			Input: "Not a date", // clearly wrong
			Panic: true,
		},
		"Case_005": {
			Input: "", // empty input
			Panic: true,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r != nil {
					if !testCase.Panic {
						t.Errorf("unexpected panic for input %q: %v", testCase.Input, r)
					}
				} else {
					if testCase.Panic {
						t.Errorf("expected panic for input %q but none occurred", testCase.Input)
					}
				}
			}()

			got := parseTimeOrPanic(testCase.Input)
			if !testCase.Panic && !got.Equal(testCase.Expected) {
				t.Errorf("expected %v, got %v", testCase.Expected, got)
			}
		})
	}
}
