package parser

import (
	"regexp"

	packrat "github.com/cphaensch/go-packrat/v2"
)

type Requisite interface{}

var ExprParser packrat.Parser[Requisite]

func init() {
	// Temp matcher functions (replace with real ones later)
	// courseMatcher := func(match string, sub []string) Requisite { return match }
	// gpaMatcher := func(match string, sub []string) Requisite { return match }
	// consentMatcher := func(match string, sub []string) Requisite { return match }

	// Course parser - Ex: "CS 1200", "MATH 2413", "PHYS 2325"
	// Original pattern: utils.Regexpf(`^\s*%s\s*$`, utils.R_SUBJ_COURSE_CAP)
	// Subgroups: [full match, subject, number]
	courseParser := packrat.NewRegexParser(
		// Callback function - receives the matched string, returns a Requisite
		func(s string) Requisite {
			// Compile the same pattern to extract subgroups
			re := regexp.MustCompile(`([A-Z]{2,4})\s*(\d{3}[A-Z]?)`)

			// FindStringSubmatch returns:
			//   sub[0] = full match (e.g., "CS 1200")
			//   sub[1] = subject (e.g., "CS")
			//   sub[2] = number (e.g., "1200")
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

	// GPA parser - Ex: "Minimum GPA of 2.5", "2.75 GPA"
	gpaParser := packrat.NewRegexParser(
		func(s string) Requisite {
			re := regexp.MustCompile(`(\d+\.\d+)`)
			sub := re.FindStringSubmatch(s)
			if len(sub) >= 2 {
				return GPAMatcher(s, sub)
			}
			return nil
		},
		`(?i)(?:minimum\s+)?gpa\s+of\s+\d+\.\d+|\d+\.\d+\s+gpa`,
		true, false,
	)

	// Consent parser - Ex: "Instructor consent required"
	consentParser := packrat.NewRegexParser(
		func(s string) Requisite {
			re := regexp.MustCompile(`(.+)\s+consent\s+required`)
			sub := re.FindStringSubmatch(s)
			if len(sub) >= 2 {
				return ConsentMatcher(s, sub)
			}
			return nil
		},
		`(?i).+\s+consent\s+required`,
		true, false,
	)

	// Combine them (order matters)
	ExprParser = packrat.NewOrParser(
		courseParser,
		gpaParser,
		consentParser,
		// rest of matchers
	)
}
