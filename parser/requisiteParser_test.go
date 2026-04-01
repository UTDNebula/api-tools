package parser

import (
	"reflect"
	"strings"
	"testing"

	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestInitMatchers(t *testing.T) {
	// Test 1: Initialization
	Matchers = nil
	initMatchers()

	if Matchers == nil {
		t.Error("Matchers should not be nil after initialization")
	}

	if Matchers == nil {
		t.Error("Matchers contain matchers after initialization")
	}

	// Test 2: No regex compilation errors
	for i, m := range Matchers {
		if m.Regex == nil {
			t.Errorf("Matcher %d has nil regex", i)
		}
	}
}

func TestGroupParens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		groups   []string
	}{
		{
			name:     "No parentheses",
			input:    "MATH 2417 and PHYS 2125",
			expected: "MATH 2417 and PHYS 2125",
			groups:   []string{},
		},
		{
			name:     "Single parentheses",
			input:    "MATH 2417 and (PHYS 2125 or PHYS 2126)",
			expected: "MATH 2417 and @0",
			groups:   []string{"PHYS 2125 or PHYS 2126"},
		},
		{
			name:     "Nested parentheses",
			input:    "((A and B) or (C and D))",
			expected: "@2",
			groups:   []string{"A and B", "C and D", "@0 or @1"},
		},
		{
			name:     "Multiple parentheses",
			input:    "(A) and (B) or (C)",
			expected: "@0 and @1 or @2",
			groups:   []string{"A", "B", "C"},
		},
		{
			name:     "Mismatched closing parentheses",
			input:    "(A and B)) extra text",
			expected: "@0) extra text",
			groups:   []string{"A and B"},
		},
		{
			name:     "Complex expression",
			input:    "MATH 2417 and (PHYS 2125 or (PHYS 2126 and CHEM 1311))",
			expected: "MATH 2417 and @1",
			groups:   []string{"PHYS 2126 and CHEM 1311", "PHYS 2125 or @0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, groups := groupParens(tt.input)
			if result != tt.expected {
				t.Errorf("groupParens() = %q, want %q", result, tt.expected)
			}
			if len(groups) != len(tt.groups) {
				t.Errorf("group count = %d, want %d", len(groups), len(tt.groups))
			}
			for i, group := range groups {
				if i < len(tt.groups) && group != tt.groups[i] {
					t.Errorf("group[%d] = %q, want %q", i, group, tt.groups[i])
				}
			}
		})
	}
}

func TestUngroupText(t *testing.T) {
	// Setup groupList as it would be after parsing
	groupList = []string{
		"PHYS 2125 or PHYS 2126",
		"A and B",
		"C and D",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No group tags",
			input:    "MATH 2417 and something",
			expected: "MATH 2417 and something",
		},
		{
			name:     "Single group tag",
			input:    "MATH 2417 and @0",
			expected: "MATH 2417 and (PHYS 2125 or PHYS 2126)",
		},
		{
			name:     "Multiple group tags",
			input:    "@1 or @2",
			expected: "(A and B) or (C and D)",
		},
		{
			name:     "Nested replacement",
			input:    "@0 with extra @1",
			expected: "(PHYS 2125 or PHYS 2126) with extra (A and B)",
		},
		{
			name:     "Out of bounds tag",
			input:    "@9",
			expected: "@9", // Should remain unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ungroupText(tt.input)
			if result != tt.expected {
				t.Errorf("ungroupText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestJoinAdjacentOthers(t *testing.T) {
	testSectionID := primitive.NewObjectID()

	tests := []struct {
		name       string
		input      []interface{}
		joinString string
		expected   []interface{}
	}{
		{
			name:       "empty input",
			input:      []interface{}{},
			joinString: ", ",
			expected:   []interface{}{},
		},
		{
			name: "single other unchanged",
			input: []interface{}{
				*schema.NewOtherRequirement("Only one", ""),
			},
			joinString: ", ",
			expected: []interface{}{
				*schema.NewOtherRequirement("Only one", ""),
			},
		},
		{
			name: "adjacent others joined",
			input: []interface{}{
				*schema.NewOtherRequirement("First", ""),
				*schema.NewOtherRequirement("Second", ""),
				*schema.NewOtherRequirement("Third", ""),
			},
			joinString: " or ",
			expected: []interface{}{
				*schema.NewOtherRequirement("First or Second or Third", ""),
			},
		},
		{
			name: "others at start and end separated by non-other",
			input: []interface{}{
				*schema.NewOtherRequirement("Start1", ""),
				*schema.NewOtherRequirement("Start2", ""),
				schema.NewSectionRequirement(testSectionID),
				*schema.NewOtherRequirement("End1", ""),
				*schema.NewOtherRequirement("End2", ""),
			},
			joinString: " and ",
			expected: []interface{}{
				*schema.NewOtherRequirement("Start1 and Start2", ""),
				schema.NewSectionRequirement(testSectionID),
				*schema.NewOtherRequirement("End1 and End2", ""),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinAdjacentOthers(tt.input, tt.joinString)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d items, got %d", len(tt.expected), len(result))
				return // Don't continue if lengths don't match
			}

			// Check each item
			for i := range result {
				switch r := result[i].(type) {
				case schema.OtherRequirement:
					expected, ok := tt.expected[i].(schema.OtherRequirement)
					if !ok {
						t.Errorf("position %d: expected OtherRequirement, got %T", i, tt.expected[i])
						continue
					}
					if r.Description != expected.Description {
						t.Errorf("position %d: description mismatch\n  got: %q\n  want: %q",
							i, r.Description, expected.Description)
					}
				case *schema.SectionRequirement:
					expected, ok := tt.expected[i].(*schema.SectionRequirement)
					if !ok {
						t.Errorf("position %d: expected *SectionRequirement, got %T", i, tt.expected[i])
						continue
					}
					if r.SectionReference != expected.SectionReference {
						t.Errorf("position %d: section reference mismatch", i)
					}
				default:
					if !reflect.DeepEqual(result[i], tt.expected[i]) {
						t.Errorf("position %d: item mismatch", i)
					}
				}
			}
		})
	}
}

func TestReqIsThrowaway(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{
			name:     "throwaway requirement returns true",
			input:    schema.Requirement{Type: "throwaway"},
			expected: true,
		},
		{
			name:     "non-throwaway requirement returns false",
			input:    schema.Requirement{Type: "course"},
			expected: false,
		},
		{
			name:     "not a requirement returns false",
			input:    "not a requirement",
			expected: false,
		},
		{
			name:     "nil returns false",
			input:    nil,
			expected: false,
		},
		{
			name:     "other requirement type returns false",
			input:    schema.Requirement{Type: "section"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reqIsThrowaway(tt.input)
			if result != tt.expected {
				t.Errorf("reqIsThrowaway(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMakeSubgroup(t *testing.T) {
	tests := []struct {
		name           string
		group          string
		subtext        string
		requisite      interface{}
		expectedGroup  string
		expectedReqLen int
	}{
		{
			name:           "basic text replacement",
			group:          "Complete MATH 2413",
			subtext:        "MATH 2413",
			requisite:      schema.CourseRequirement{ClassReference: "MATH 2413"},
			expectedGroup:  "Complete @0",
			expectedReqLen: 1,
		},
		{
			name:    "choice requisite replacement",
			group:   "Credit cannot be received for both courses, @0 and @1",
			subtext: "Credit cannot be received for both courses, @0 and @1",
			requisite: schema.ChoiceRequirement{
				Requirement: schema.Requirement{Type: "choice"},
				Choices: &schema.CollectionRequirement{
					Requirement: schema.Requirement{Type: "collection"},
					Name:        "",
					Required:    1,
					Options: []interface{}{
						schema.NewCourseRequirement("CSCI0190", ""),
						schema.NewCourseRequirement("CSCI0200", ""),
					},
				},
			},
			expectedGroup:  "@2",
			expectedReqLen: 3,
		},
		{
			name:           "core requirement replacement",
			group:          "Completion of an 010 core course",
			subtext:        "Completion of an 010 core course",
			requisite:      schema.CourseRequirement{ClassReference: "010", MinimumGrade: "B"},
			expectedGroup:  "@0",
			expectedReqLen: 1,
		},
		{
			name:           "multiple occurrence replacement",
			group:          "CS 2336 and CS 2336 lab",
			subtext:        "CS 2336",
			requisite:      schema.CourseRequirement{ClassReference: "CS 2336"},
			expectedGroup:  "@0 and @0 lab",
			expectedReqLen: 1,
		},
		{
			name:           "subtext not found",
			group:          "Complete CHEM 1311",
			subtext:        "PHYS 2325",
			requisite:      schema.CourseRequirement{ClassReference: "PHYS 2325"},
			expectedGroup:  "Complete CHEM 1311",
			expectedReqLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global variables
			requisiteList = []interface{}{}
			groupList = []string{}

			// For the choice requisite test, pre-populate with 2 dummy requisites
			if tt.name == "choice requisite replacement" {
				requisiteList = []interface{}{
					schema.CourseRequirement{ClassReference: "DUMMY1", MinimumGrade: ""},
					schema.CourseRequirement{ClassReference: "DUMMY2", MinimumGrade: ""},
				}
			}

			// Call the function
			result := makeSubgroup(tt.group, tt.subtext, tt.requisite)

			// Check the returned group
			if result != tt.expectedGroup {
				t.Errorf("expected group %q, got %q", tt.expectedGroup, result)
			}

			// Check the requisite list length
			if len(requisiteList) != tt.expectedReqLen {
				t.Errorf("expected %d requisites, got %d", tt.expectedReqLen, len(requisiteList))
			}

			// Check if requisite was added
			if tt.subtext != "" && strings.Contains(tt.group, tt.subtext) && len(requisiteList) > 0 {
				if requisiteList[len(requisiteList)-1] != tt.requisite {
					t.Errorf("requisite not properly added to list")
				}
			}

			// Check if group was added to groupList
			if len(groupList) != 1 {
				t.Errorf("expected 1 group in groupList, got %d", len(groupList))
			} else if groupList[0] != result {
				t.Errorf("groupList doesn't contain the result")
			}
		})
	}
}

func TestFindICN(t *testing.T) {
	// Save original Courses and ensure cleanup, even if there is error
	originalCourses := Courses
	t.Cleanup(func() {
		Courses = originalCourses
	})

	// Create test course
	testCourse := &schema.Course{
		Subject_prefix:         "CS",
		Course_number:          "1337",
		Internal_course_number: "ICN001",
	}

	tests := []struct {
		name    string
		setup   map[string]*schema.Course
		subject string
		number  string
		wantICN string
		wantErr bool
	}{
		{
			name: "finds existing course",
			setup: map[string]*schema.Course{
				"": testCourse,
			},
			subject: "CS",
			number:  "1337",
			wantICN: "ICN001",
			wantErr: false,
		},
		{
			name: "course not found returns error",
			setup: map[string]*schema.Course{
				"": testCourse,
			},
			subject: "MATH",
			number:  "101",
			wantICN: "ERROR",
			wantErr: true,
		},
		{
			name:    "empty courses list returns error",
			setup:   map[string]*schema.Course{},
			subject: "CS",
			number:  "1337",
			wantICN: "ERROR",
			wantErr: true,
		},
		{
			name: "case sensitive match fails",
			setup: map[string]*schema.Course{
				"": testCourse,
			},
			subject: "cs", // lowercase
			number:  "1337",
			wantICN: "ERROR",
			wantErr: true,
		},
		{
			name: "multiple courses finds correct one",
			setup: map[string]*schema.Course{
				"": testCourse,
				".": {
					Subject_prefix:         "MATH",
					Course_number:          "101",
					Internal_course_number: "ICN002",
				},
			},
			subject: "MATH",
			number:  "101",
			wantICN: "ICN002",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test data
			Courses = tt.setup

			gotICN, err := findICN(tt.subject, tt.number)

			// Check error
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Check ICN
			if gotICN != tt.wantICN {
				t.Errorf("got ICN %q, want %q", gotICN, tt.wantICN)
			}
		})
	}

}
