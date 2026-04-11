package parser

import (
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	packrat "github.com/cphaensch/go-packrat/v2"
)

var whitespace = regexp.MustCompile(`\s+`)

func TestMinimal(t *testing.T) {
	log.SetFlags(log.Flags() | utils.Lverbose)
	// Add mock courses so findICN succeeds
	Courses = map[string]*schema.Course{
		"CS 1200":   {Subject_prefix: "CS", Course_number: "1200", Internal_course_number: "CS-1200"},
		"MATH 2413": {Subject_prefix: "MATH", Course_number: "2413", Internal_course_number: "MATH-2413"},
	}
	t.Cleanup(func() { Courses = nil })

	tests := []struct {
		input    string
		wantType string
		wantNil  bool
	}{
		// Basic parsers
		{"CS 1200", "*schema.CourseRequirement", false},
		{"Minimum GPA of 3.0", "*schema.GPARequirement", false},
		{"Instructor consent required", "*schema.ConsentRequirement", false},

		// The subgroup-index cases
		{"CS majors only", "*schema.MajorRequirement", false},
		{"CS minors only", "*schema.MinorRequirement", false},
		{"CS majors and minors only", "*schema.CollectionRequirement", false},
		{"Completion of 130 core course", "*schema.CoreRequirement", false},

		// AND/OR
		// {"CS 1200 and MATH 2413", "schema.CollectionRequirement", false},
		// {"CS 1200 or MATH 2413", "*schema.CollectionRequirement", false},

		// Fallback
		{"same as CS 1200", "schema.Requirement", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			scanner := packrat.NewScanner[Requisite](tt.input, whitespace)
			node, ok := ExprParser.Match(scanner)

			if !ok {
				t.Errorf("Match() failed for %q", tt.input)
			}

			if node.Payload == nil && !tt.wantNil {
				t.Errorf("Payload is nil for %q", tt.input)
			}

			gotType := ""
			if node.Payload != nil {
				gotType = getTypeName(node.Payload)
				if gotType != tt.wantType {
					t.Errorf("Type = %s, want %s", gotType, tt.wantType)
				}
			}
		})
	}
}

func getTypeName(i interface{}) string {
	return fmt.Sprintf("%T", i)
}
