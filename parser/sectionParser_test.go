package parser

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

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

			defer FailTestIfNoPanic(t, name)
			getInternalClassAndCourseNum(fail)

		})
	}
}

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

			defer FailTestIfNoPanic(t, name)
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

			if testCase.Panic {
				defer FailTestIfNoPanic(t, name)
			} else {
				defer FailTestIfPanic(t, name)
			}

			got := parseTimeOrPanic(testCase.Input)
			if !testCase.Panic && !got.Equal(testCase.Expected) {
				t.Errorf("expected %v, got %v", testCase.Expected, got)
			}
		})
	}
}
