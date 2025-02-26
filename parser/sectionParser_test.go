package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetInternalClassAndCourseNum(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
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
}

func TestGetAcademicSession(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
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
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getSectionNumber(testCase.ClassInfo)
			expected := testCase.Section.Section_number

			if output != expected {
				t.Errorf("expected %s got %s", expected, output)
			}
		})

	}
}

func TestGetTeachingAssistants(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
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
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getInstructionMode(testCase.ClassInfo)
			expected := testCase.Section.Instruction_mode

			if output != expected {
				t.Errorf("expected %s got %s", expected, output)
			}
		})

	}
}

func TestGetMeetings(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
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
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
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
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getSyllabusUri(testCase.RowInfo)
			expected := testCase.Section.Syllabus_uri

			if output != expected {
				t.Errorf("expected %s got %s", expected, output)
			}
		})

	}
}
