package parser

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"strings"
	"testing"
)

func TestGetInternalClassAndCourseNum(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			classNum, _ := getInternalClassAndCourseNum(testCase.ClassInfo)
			expectedClassNum := testCase.Section.Internal_class_number
			//expectedCourseNum := Courses[CourseIDMap[testCase.Section.Course_reference]].Course_number

			if classNum != expectedClassNum {
				t.Errorf("expected %v got %v\n", expectedClassNum, classNum)
			}
			/*
				if courseNum != expectedCourseNum {
					t.Errorf("expected %v got %v\n", expectedCourseNum, courseNum)
				}

			*/
		})
	}
}

func TestGetAcademicSession(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getAcademicSession(testCase.RowInfo)
			expected := testCase.Section.Academic_session

			var diffBuilder strings.Builder
			if output.Name != expected.Name {
				diffBuilder.WriteString(fmt.Sprintf("Name: expected %v got %v\n", expected.Name, output.Name))
			}
			if !output.Start_date.Equal(expected.Start_date) {
				diffBuilder.WriteString(fmt.Sprintf("Start_Date: expected %v got %v\n", expected.Start_date, output.Start_date))
			}
			if !output.End_date.Equal(expected.End_date) {
				diffBuilder.WriteString(fmt.Sprintf("End_Date: expected %v got %v\n", expected.End_date, output.End_date))
			}

			if diffBuilder.Len() > 0 {
				t.Errorf("Failed to parse Adacemic Session\n%s\n", diffBuilder.String())
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
				t.Errorf("expected %v got %v\n", expected, output)
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

				var diffBuilder strings.Builder
				if assistant.First_name != expectedAssistant.First_name {
					diffBuilder.WriteString(fmt.Sprintf("First_name: expected %v got %v\n", assistant.First_name, expectedAssistant.First_name))
				}
				if assistant.Last_name != expectedAssistant.Last_name {
					diffBuilder.WriteString(fmt.Sprintf("Last_name: expected %v got %v\n", assistant.Last_name, expectedAssistant.Last_name))
				}
				if assistant.Role != expectedAssistant.Role {
					diffBuilder.WriteString(fmt.Sprintf("Role: expected %v got %v\n", assistant.Role, expectedAssistant.Role))
				}
				if assistant.Email != expectedAssistant.Email {
					diffBuilder.WriteString(fmt.Sprintf("Email: expected %v got %v\n", assistant.Email, expectedAssistant.Email))
				}

				if diffBuilder.Len() > 0 {
					t.Errorf("Failed to parse Adacemic Session\n%s\n", diffBuilder.String())
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

			if output != expected {
				t.Errorf("expected %v got %v\n", expected, output)
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
				if flag != expectedFlag {
					t.Errorf("expected %v got %v\n", expected, output)
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

			if output != expected {
				t.Errorf("expected %v got %v\n", expected, output)
			}
		})

	}
}

/* There is a issue loading grade data, it cant create log file

func TestGetGradeDistribution(t *testing.T) {
	loadTestData(t)

	GradeMap = loadGrades("../grade-data/")

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			courseRef := Courses[CourseIDMap[testCase.Section.Course_reference]]
			output := getGradeDistribution(testCase.Section.Academic_session, testCase.Section.Section_number, courseRef)
			expected := testCase.Section.Grade_distribution

			if !reflect.DeepEqual(output, expected) {
				t.Errorf("expected %d meetings got %d", len(expected), len(output))
				return
			}
		})
	}
}
*/
