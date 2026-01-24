package parser

import (
	"testing"
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
