package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	packrat "github.com/cphaensch/go-packrat/v2"
)

var Whitespace = regexp.MustCompile(`\s+`)

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

// ParseRequirement is the main entry point for parsing requirement text.
// It handles AND/OR at the top level by splitting the string, then uses
// the packrat parser for the individual parts.
func ParseRequirement(text string) *schema.CollectionRequirement {
	text = strings.TrimSpace(text)

	// TODO: I should probably handle paranthesis here, but recursion makes me sad

	// Handle AND chains
	if strings.Contains(text, " and ") {
		parts := strings.Split(text, " and ")
		options := make([]interface{}, 0, len(parts))
		for _, part := range parts {
			req := parseLeaf(part)
			if req != nil && !reqIsThrowaway(req) {
				options = append(options, req)
			}
		}
		// AND means all options are required
		return schema.NewCollectionRequirement("", len(options), options)
	}

	// Handle OR chains
	if strings.Contains(text, " or ") {
		parts := strings.Split(text, " or ")
		options := make([]interface{}, 0, len(parts))
		for _, part := range parts {
			req := parseLeaf(part)
			if req != nil && !reqIsThrowaway(req) {
				options = append(options, req)
			}
		}
		// OR means at least one option is required
		return schema.NewCollectionRequirement("", 1, options)
	}

	// No AND/OR – parse as a single leaf requirement
	leaf := parseLeaf(text)
	if leaf == nil || reqIsThrowaway(leaf) {
		return nil
	}
	// Wrap as a collection with one option, required = 1
	return schema.NewCollectionRequirement("", 1, []interface{}{leaf})
}

// parseLeaf uses the packrat parser (which now contains only leaf parsers) to parse a single atomic requirement.
func parseLeaf(input string) interface{} {
	scanner := packrat.NewScanner[Requisite](input, Whitespace)
	node, ok := ExprParser.Match(scanner)
	if ok && node.Payload != nil {
		return node.Payload
	}
	return nil
}

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
				utils.VPrintf("[ToParser] Converting %s to parser", d.Name)
				re := regexp.MustCompile(pat)
				sub := re.FindStringSubmatch(s)
				if len(sub) >= d.GroupCount {
					utils.VPrintf("[ToParser] Match success: input=%q groups=%v", s, sub)
					return d.Matcher(s, sub)
				}
				utils.VPrintf("[ToParser] Match fail: input=%q (groups=%d, need %d)", s, len(sub), d.GroupCount)
				return nil
			},
			pat, d.SkipWhitespace, d.CaseSensitive,
		)
		subParsers = append(subParsers, p)
	}

	return packrat.NewOrParser(subParsers...)
}

func init() {
	Register(&Definition{
		Name:           "ThrowawayParser",
		Patterns:       []string{`^(?i)(?:better|\d-\d|same as.+)$`},
		GroupCount:     1,
		Matcher:        ThrowawayMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name: "OtherParser",
		Patterns: []string{
			fmt.Sprintf(`(?i).+%s\s+only$`, utils.R_YEARS), // * <YEAR> Only
			`(?i).+\s+in\s+any\s+combination\s+of\s+.+`,    // * in any combination of *
		},
		GroupCount:     1,
		Matcher:        OtherMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name: "MajorMinorParser",
		Patterns: []string{
			// <SUBJECT> majors and minors only
			fmt.Sprintf(`(?i)((%s)\s+majors\s+and\s+minors\s+only)`, utils.R_SUBJECT),
		},
		GroupCount:     3,
		Matcher:        MajorMinorMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name: "CoreCompletionParser",
		Patterns: []string{
			// Completion of [a/an] <CORE CODE> core [course]
			`(?i)(Completion\s+of\s+(?:an?\s+)?(\d{3}).+core(?:\s+course)?)`,
		},
		GroupCount:     3,
		Matcher:        CoreCompletionMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name: "ChoiceParser",
		Patterns: []string{
			// Credit cannot be received for both [courses][,] <EXPRESSION>
			`(?i)(Credit\s+cannot\s+be\s+received\s+for\s+both\s+(?:courses)?,?(.+))`,
			// Credit cannot be received for both [courses][,] <EXPRESSION>
			`(?i)(Credit\s+cannot\s+be\s+received\s+for\s+more\s+than\s+one\s+of.+:(.+))`,
		},
		GroupCount:     3,
		Matcher:        ChoiceMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name: "SubstitionParser",
		Patterns: []string{
			fmt.Sprintf(`^(?i)(%s\s+with\s+a(?:\s+grade)?(?:\s+of)?\s+(%s)\s+or\s+better)`, utils.R_SUBJ_COURSE_CAP, utils.R_GRADE), // [name, number, min grade]
		},
		GroupCount:     4,
		Matcher:        SubstitutionMatcher(CourseMinGradeMatcher),
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name: "CourseMinGradeParser",
		Patterns: []string{
			//<COURSE> with a [minimum] grade of [at least] [a] <GRADE>
			fmt.Sprintf(`^(?i)%s\s+with\s+a\s+(?:minimum\s+)?grade\s+of\s+(?:at least\s+)?(?:a\s+)?(%s)$`, utils.R_SUBJ_COURSE_CAP, utils.R_GRADE), // [name, number, min grade]
			// A grade of [at least] [a] <GRADE> in <COURSE>
			fmt.Sprintf(`^(?i)A\s+grade\s+of(?:\s+at\s+least)?(?:\s+a)?\s+(%s)\s+in\s+%s$`, utils.R_GRADE, utils.R_SUBJ_COURSE_CAP), // [min grade, name, number]
		},
		GroupCount:     3,
		Matcher:        CourseMinGradeMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name:           "CourseParser",
		Patterns:       []string{fmt.Sprintf(`^\s*%s\s*$`, utils.R_SUBJ_COURSE_CAP)},
		GroupCount:     3,
		Matcher:        CourseMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name:           "ConsentParser",
		Patterns:       []string{`^(?i)(.+)\s+consent\s+required`},
		GroupCount:     2,
		Matcher:        ConsentMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name:           "LimitParser",
		Patterns:       []string{`^(?i)(\d+)\s+semester\s+credit\s+hours\s+maximum$`},
		GroupCount:     2,
		Matcher:        LimitMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	// <SUBJECT> majors only
	Register(&Definition{
		Name:           "MajorParser",
		Patterns:       []string{`^(?i)(.+)\s+major(?:s\s+only)?$`},
		GroupCount:     2,
		Matcher:        MajorMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	// <SUBJECT> minors only
	Register(&Definition{
		Name:           "MinorParser",
		Patterns:       []string{`^(?i)(.+)\s+minor(?:s\s+only)?$`},
		GroupCount:     2,
		Matcher:        MinorMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name: "CoreParser",
		Patterns: []string{
			// Any <HOURS> semester credit hour <CORE> course
			`^(?i)any\s+(\d+)\s+semester\s+credit\s+hour\s+(\d{3})(?:\s+@\d+)?\s+core(?:\s+course)?$`,
		},
		GroupCount:     3,
		Matcher:        CoreMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name: "GPAParser",
		Patterns: []string{
			// Minimum GPA of <GPA>
			`^(?i)([0-9\.]+) GPA$`, // [GPA]
			// Minimum GPA of <GPA>
			`^(?i)(?:minimum\s+)?GPA\s+of\s+([0-9\.]+)$`,
			// A university grade point average of at least <GPA>
			`^(?i)a(?:\s+university)?\s+grade\s+point\s+average\s+of(?:\s+at\s+least)?\s+([0-9\.]+)$`,
		},
		GroupCount:     2,
		Matcher:        GPAMatcher,
		SkipWhitespace: true,
		CaseSensitive:  false,
	})

	Register(&Definition{
		Name:           "GroupParser",
		Patterns:       []string{`@(\d+)`},
		GroupCount:     2,
		Matcher:        GroupTagMatcher,
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
