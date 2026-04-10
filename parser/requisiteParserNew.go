package parser

import (
	"fmt"
	"regexp"

	"github.com/UTDNebula/api-tools/utils"
	packrat "github.com/cphaensch/go-packrat/v2"
)

type Requisite interface{}

var ExprParser packrat.Parser[Requisite]

type Definition struct {
	Name           string
	Patterns       []string
	GroupCount     int
	Matcher        func(group string, subgroups []string) interface{}
	SkipWhitespace bool
	CaseSensitive  bool
}

var registry []*Definition

// TODO: Probably check values first before appending
func Register(d *Definition) {
	registry = append(registry, d)
}

func (d *Definition) ToParser() packrat.Parser[Requisite] {
	// Build an OrParser over each pattern
	var subParsers []packrat.Parser[Requisite]

	for _, pat := range d.Patterns {
		p := packrat.NewRegexParser(
			func(s string) Requisite {
				re := regexp.MustCompile(pat)
				sub := re.FindStringSubmatch(s)
				if len(sub) >= d.GroupCount {
					return d.Matcher(s, sub)
				}
				return nil
			},
			pat, d.SkipWhitespace, d.CaseSensitive,
		)
		subParsers = append(subParsers, p)
	}

	return packrat.NewOrParser(subParsers...)
}

func init() {
	/*
		// Course parser - Ex: "CS 1200", "MATH 2413", "PHYS 2325"
		// Original pattern: utils.Regexpf(`^\s*%s\s*$`, utils.R_SUBJ_COURSE_CAP)
		// Subgroups: [full match, subject, number]
		courseParser := packrat.NewRegexParser(
			// Callback function - receives the matched string, returns a Requisite
			func(s string) Requisite {
				// Compile the same pattern to extract subgroups
				re := regexp.MustCompile(`([A-Z]{2,4})\s*(\d{3}[A-Z]?)`)

				// FindStringSubmatch returns:
				//   sub[0] = full match (Ex: "CS 1200")
				//   sub[1] = subject (Ex: "CS")
				//   sub[2] = number (Ex: "1200")
				sub := re.FindStringSubmatch(s)

				// Need at least 3 elements (full match + 2 capture groups)
				if len(sub) >= 3 {
					// Pass sub which is [full match, subject, number] to the matcher
					// This matches the original CourseMatcher(group, subgroups) signature
					return CourseMatcher(s, sub)
				}
				return nil
			},
			// Pattern for packrat to match against
			`[A-Z]{2,4}\s*\d{3}[A-Z]?`,
			true,  // skipWhitespace - packrat automatically skips whitespace between tokens
			false, // caseSensitive - false means "CS" and "cs" both match
		)
	*/

	Register(&Definition{
		Name: "CourseParser",
		Patterns: []string{
			fmt.Sprintf(`^\s*%s\s*$`, utils.R_SUBJ_COURSE_CAP), // [name, number]
		},
		Matcher:        CourseMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name: "GPAParser",
		Patterns: []string{
			`(\d+\.\d+)`,
			`^(?i)(?:minimum\s+)?GPA\s+of\s+([0-9\.]+)$`,
			`^(?i)a(?:\s+university)?\s+grade\s+point\s+average\s+of(?:\s+at\s+least)?\s+([0-9\.]+)$`,
		},
		Matcher:        GPAMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name:           "ConsentParser",
		Patterns:       []string{`^(?i)(.+)\s+consent\s+required`},
		Matcher:        ConsentMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	var parsers []packrat.Parser[Requisite]
	for _, p := range registry {
		parsers = append(parsers, p.ToParser())
	}

	// Combine them (note order matters)
	ExprParser = packrat.NewOrParser(
		parsers...,
	)
}
