package parser

import (
	"reflect"
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
