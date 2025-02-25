package parser

import (
	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"testing"
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

/* todo fix
func TestGetCatalogYear(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getCatalogYear(testCase.Section.Academic_session)
			expected := testCase.Section.Academic_session

			if diffBuilder.Len() > 0 {
				t.Errorf("Failed to parse Adacemic Session\n%s\n", diffBuilder.String())
			}
		})

	}
}

*/
/*

func TestGetInternalCourseNumber(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getInternalCourseNumber(testCase.ClassInfo)
			expected := testCase.Course.Internal_course_number

			if expected != output {
				t.Errorf("expected %v got %v", expected, output)
			}
		})
	}
}

func TestGetTitle(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getTitle(testCase.RowInfo)
			expected := testCase.Course.Title

			if expected != output {
				t.Errorf("expected %v got %v", expected, output)
			}
		})
	}
}

func TestGetDescription(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getDescription(testCase.RowInfo)
			expected := testCase.Course.Description

			if expected != output {
				t.Errorf("expected %v got %v", expected, output)
			}
		})
	}
}

func TestGetSchool(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getSchool(testCase.RowInfo)
			expected := testCase.Course.School

			if expected != output {
				t.Errorf("expected %v got %v", expected, output)
			}
		})
	}
}

func TestCreditHours(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getCreditHours(testCase.ClassInfo)
			expected := testCase.Course.Credit_hours

			if expected != output {
				t.Errorf("expected %v got %v", expected, output)
			}
		})
	}
}

func TestGetClassLevel(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getClassLevel(testCase.ClassInfo)
			expected := testCase.Course.Class_level

			if expected != output {
				t.Errorf("expected %v got %v", expected, output)
			}
		})
	}
}

func TestGetActivityType(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getActivityType(testCase.ClassInfo)
			expected := testCase.Course.Activity_type

			if expected != output {
				t.Errorf("expected %v got %v", expected, output)
			}
		})
	}

}

func TestGetGrading(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			output := getGrading(testCase.ClassInfo)
			expected := testCase.Course.Grading

			if expected != output {
				t.Errorf("expected %v got %v", expected, output)
			}
		})
	}

}

func TestGetCourseNumberAndPrefix(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			outputNumber, outputPrefix := getCourseNumberAndPrefix(testCase.ClassInfo)
			expectedNumber, expectedPrefix := testCase.Course.Course_number, testCase.Course.Subject_prefix

			if outputNumber != expectedNumber {
				t.Errorf("expected %v got %v", expectedNumber, outputNumber)
			}
			if outputPrefix != expectedPrefix {
				t.Errorf("expected %v got %v", expectedPrefix, outputPrefix)
			}
		})
	}

}

func TestGetContactMatches(t *testing.T) {
	loadTestData(t)

	for name, testCase := range testDataCache {
		t.Run(name, func(t *testing.T) {
			lectureHours, labHours, offeringFrequency := getContactMatches(testCase.Course.Description)
			expectedLectureHours := testCase.Course.Lecture_contact_hours
			expectedLabHours := testCase.Course.Laboratory_contact_hours
			expectedOfferingFrequency := testCase.Course.Offering_frequency

			if lectureHours != expectedLectureHours {
				t.Errorf("expected %v got %v", expectedLectureHours, lectureHours)
			}
			if labHours != expectedLabHours {
				t.Errorf("expected %v got %v", expectedLabHours, labHours)
			}
			if offeringFrequency != expectedOfferingFrequency {
				t.Errorf("expected %v got %v", expectedOfferingFrequency, offeringFrequency)
			}
		})
	}

}

*/
