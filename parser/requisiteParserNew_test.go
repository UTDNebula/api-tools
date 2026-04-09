package parser

import (
	"regexp"
	"testing"

	packrat "github.com/cphaensch/go-packrat/v2"
)

var whitespace = regexp.MustCompile(`\s+`)

func TestMinimal(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"CS 1200", true},
		{"MATH 2413", true},
		{"Minimum GPA of 3.0", true},
		{"2.75 GPA", true},
		{"Instructor consent required", true},
		{"gibberish", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			scanner := packrat.NewScanner[Requisite](tt.input, whitespace)
			node, ok := ExprParser.Match(scanner)

			if ok != tt.want {
				t.Errorf("Match() = %v, want %v", ok, tt.want)
			}
			if ok && node.Payload == nil {
				t.Error("Match succeeded but payload is nil")
			}
		})
	}
}
