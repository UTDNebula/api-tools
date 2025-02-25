package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetInternalClassAndCourseNum(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			classNum, _ := getInternalClassAndCourseNum(testCase.ClassInfo)
			expectedClassNum := testCase.Section.Internal_class_number

			diff := cmp.Diff(expectedClassNum, classNum)

			if diff != "" {
				t.Errorf("Failed (-expected +got)\n %s", diff)
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

			diff := cmp.Diff(expected, output)

			if diff != "" {
				t.Errorf("Failed (-expected +got)\n %s", diff)
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

			if len(output) != len(expected) {
				t.Errorf("expected %d assistants got %d", len(expected), len(output))
				return
			}

			for i, assistant := range output {
				expectedAssistant := expected[i]

				diff := cmp.Diff(expectedAssistant, assistant)

				if diff != "" {
					t.Errorf("Failed (-expected +got)\n %s", diff)
				}

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

			diff := cmp.Diff(expected, output)

			if diff != "" {
				t.Errorf("Failed (-expected +got)\n %s", diff)
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

			if len(output) != len(expected) {
				t.Errorf("expected %d meetings got %d", len(expected), len(output))
				return
			}

			for i, meeting := range output {
				expectedMeeting := expected[i]

				diff := cmp.Diff(meeting, expectedMeeting)

				if diff != "" {
					t.Errorf("Failed (-expected +got)\n %s", diff)
				}
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

			if len(output) != len(expected) {
				t.Errorf("expected %d meetings got %d", len(expected), len(output))
				return
			}

			for i, flag := range output {
				expectedFlag := expected[i]

				diff := cmp.Diff(expectedFlag, flag)

				if diff != "" {
					t.Errorf("Failed (-expected +got)\n %s", diff)
				}
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

			diff := cmp.Diff(expected, output)

			if diff != "" {
				t.Errorf("Failed (-expected +got)\n %s", diff)
			}
		})

	}
}
