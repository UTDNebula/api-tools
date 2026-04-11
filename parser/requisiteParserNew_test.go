package parser

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/UTDNebula/nebula-api/api/schema"
	packrat "github.com/cphaensch/go-packrat/v2"
)

var whitespace = regexp.MustCompile(`\s+`)

func TestLeafParsers(t *testing.T) {
	// Add mock courses so findICN succeeds
	Courses = map[string]*schema.Course{
		"CS 1200":   {Subject_prefix: "CS", Course_number: "1200", Internal_course_number: "CS-1200"},
		"MATH 2413": {Subject_prefix: "MATH", Course_number: "2413", Internal_course_number: "MATH-2413"},
	}

	leafTests := []struct {
		input    string
		wantType string
	}{
		{"CS 1200", "*schema.CourseRequirement"},
		{"Minimum GPA of 3.0", "*schema.GPARequirement"},
		{"Instructor consent required", "*schema.ConsentRequirement"},
		{"CS majors only", "*schema.MajorRequirement"},
		{"CS minors only", "*schema.MinorRequirement"},
		{"Completion of 130 core course", "*schema.CoreRequirement"},
		// HACK: still very confused on why throwaway returns a regular requirement
		{"same as CS 1200", "schema.Requirement"},
	}

	for _, tt := range leafTests {
		t.Run(tt.input, func(t *testing.T) {
			scanner := packrat.NewScanner[Requisite](tt.input, whitespace)
			node, ok := ExprParser.Match(scanner)
			if !ok {
				t.Errorf("Match failed")
			}
			if node.Payload == nil {
				t.Errorf("Payload nil")
			}
			gotType := getTypeName(node.Payload)
			if gotType != tt.wantType {
				t.Errorf("Type = %s, want %s", gotType, tt.wantType)
			}
		})
	}
}

func TestCollectionRequirements(t *testing.T) {
	// Add mock courses so findICN succeeds
	Courses = map[string]*schema.Course{
		"CS 1200":   {Subject_prefix: "CS", Course_number: "1200", Internal_course_number: "CS-1200"},
		"MATH 2413": {Subject_prefix: "MATH", Course_number: "2413", Internal_course_number: "MATH-2413"},
	}

	tests := []struct {
		input           string
		required        int
		optionsLen      int
		firstOptionType string
	}{
		{"CS 1200", 1, 1, "*schema.CourseRequirement"},
		{"CS 1200 and MATH 2413", 2, 2, "*schema.CourseRequirement"},
		{"CS 1200 or MATH 2413", 1, 2, "*schema.CourseRequirement"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseRequirement(tt.input)
			if result == nil {
				t.Fatal("nil result")
			}
			if result.Required != tt.required {
				t.Errorf("Required = %d, want %d", result.Required, tt.required)
			}
			if len(result.Options) != tt.optionsLen {
				t.Errorf("Options len = %d, want %d", len(result.Options), tt.optionsLen)
			}
			if tt.firstOptionType != "" && len(result.Options) > 0 {
				if getTypeName(result.Options[0]) != tt.firstOptionType {
					t.Errorf("First option type mismatch")
				}
			}
		})
	}
}

func getTypeName(i interface{}) string {
	return fmt.Sprintf("%T", i)
}
