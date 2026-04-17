package parser

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	packrat "github.com/cphaensch/go-packrat/v2"
)

type Requisite interface{}

type Definition struct {
	Name           string
	Patterns       []string
	GroupCount     int
	Matcher        func(group string, subgroups []string) interface{}
	SkipWhitespace bool
	CaseSensitive  bool
}

var registry []*Definition
var ExprParser packrat.Parser[Requisite]

// ParseRequirement is the main entry point for parsing requirement text.
// It handles AND/OR at the top level by splitting the string, then uses
// the packrat parser for the individual parts.
func ParseRequirement(text string) *schema.CollectionRequirement {
	fmt.Printf("Parsing input: %q\n", text) // Add this at function start
	text = strings.TrimSpace(text)

	flatText, groups := groupParens(text)
	groupReqs := make([]interface{}, len(groups))
	// Parse groups bottom‑up (they may contain nested @N references)
	for i := len(groups) - 1; i >= 0; i-- {
		req := ParseRequirement(groups[i]) // recursive call
		if req != nil {
			groupReqs[i] = req
		}
	}
	text = flatText

	// Handle AND chains
	if strings.Contains(text, " and ") {
		parts := strings.Split(text, " and ")
		options := make([]interface{}, 0, len(parts))
		for _, part := range parts {
			var req interface{}
			if placeholder := getPlaceholder(part, groupReqs); placeholder != nil {
				req = placeholder
			} else {
				req = parseLeaf(part)
			}
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
			var req interface{}
			if placeholder := getPlaceholder(part, groupReqs); placeholder != nil {
				req = placeholder
			} else {
				req = parseLeaf(part)
			}
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

func getPlaceholder(part string, groupReqs []interface{}) interface{} {
	if strings.HasPrefix(part, "@") && len(part) > 1 {
		idx, err := strconv.Atoi(part[1:])
		if err == nil && idx >= 0 && idx < len(groupReqs) {
			return groupReqs[idx]
		}
	}
	return nil
}

// parseLeaf uses the packrat parser (which now contains only leaf parsers) to parse a single atomic requirement.
func parseLeaf(input string) interface{} {
	scanner := packrat.NewScanner[Requisite](input, packrat.SkipWhitespaceRegex)
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

/* --- Matchers and other functions --- */

// Matcher defines a regex-driven handler used during requisite group parsing.
type Matcher struct {
	Regex   *regexp.Regexp
	Handler func(string, []string) interface{}
}

// Matchers contains the ordered collection of matcher rules applied during requisite parsing.
// NOTE: PARENTHESES ARE OF HIGHEST PRECEDENCE! (This is due to groupParens() handling grouping of parenthesized text before parsing begins)
var Matchers []Matcher

// SubstitutionMatcher returns a matcher that replaces a subgroup with parseFnc's result before parsing the outer group.
// For example, "(OPRE 3360 or STAT 3360 or STAT 4351), and JSOM majors and minors only" becomes "... and @N".
func SubstitutionMatcher(parseFnc func(string, []string) interface{}) func(string, []string) interface{} {
	// Return a closure that uses parseFnc to substitute subgroups[1]
	return func(group string, subgroups []string) interface{} {
		// If there's no text to substitute, just return an OtherRequirement
		if len(subgroups) < 2 {
			return OtherMatcher(group, subgroups)
		}
		// Otherwise, substitute subgroups[1] and parse it with parseFnc
		return parseGroup(makeSubgroup(group, subgroups[1], parseFnc(group, subgroups)))
	}
}

// CourseMinGradeMatcher returns a course requirement enforcing a minimum grade when an ICN is found.
func CourseMinGradeMatcher(group string, subgroups []string) interface{} {
	icn, err := findICN(subgroups[1], subgroups[2])
	if err != nil {
		log.Printf("WARN: %s", err)
		return OtherMatcher(group, subgroups)
	}
	return schema.NewCourseRequirement(icn, subgroups[3])
}

// CourseMatcher returns a course requirement with the default minimum grade expectation.
func CourseMatcher(group string, subgroups []string) interface{} {
	icn, err := findICN(subgroups[1], subgroups[2])
	if err != nil {
		log.Printf("WARN: %s", err)
		return OtherMatcher(group, subgroups)
	}
	return schema.NewCourseRequirement(icn, "D")
}

// ConsentMatcher captures grantor consent requirements from requisite text.
func ConsentMatcher(group string, subgroups []string) interface{} {
	return schema.NewConsentRequirement(subgroups[1])
}

// LimitMatcher produces a limit requirement that caps allowable credit hours.
func LimitMatcher(group string, subgroups []string) interface{} {
	hourLimit, err := strconv.Atoi(subgroups[1])
	if err != nil {
		panic(err)
	}
	return schema.NewLimitRequirement(hourLimit)
}

// MajorMatcher produces a major-specific requirement.
func MajorMatcher(group string, subgroups []string) interface{} {
	return schema.NewMajorRequirement(subgroups[1])
}

// MinorMatcher produces a minor-specific requirement.
func MinorMatcher(group string, subgroups []string) interface{} {
	return schema.NewMinorRequirement(subgroups[1])
}

// MajorMinorMatcher builds an OR collection spanning both major and minor requirements.
func MajorMinorMatcher(group string, subgroups []string) interface{} {
	return schema.NewCollectionRequirement("OR", 1, []interface{}{*schema.NewMajorRequirement(subgroups[1]), *schema.NewMinorRequirement(subgroups[1])})
}

// CoreMatcher creates a requirement for completion of a specific core course count.
func CoreMatcher(group string, subgroups []string) interface{} {
	hourReq, err := strconv.Atoi(subgroups[1])
	if err != nil {
		panic(err)
	}
	return schema.NewCoreRequirement(subgroups[2], hourReq)
}

// CoreCompletionMatcher indicates completion of a specific core category without an hour requirement.
func CoreCompletionMatcher(group string, subgroups []string) interface{} {
	return schema.NewCoreRequirement(subgroups[1], -1)
}

// ChoiceMatcher converts a subgroup collection into a mutually exclusive choice requirement.
func ChoiceMatcher(group string, subgroups []string) interface{} {
	collectionReq, ok := parseGroup(subgroups[1]).(*schema.CollectionRequirement)
	if !ok {
		log.Printf("WARN: ChoiceMatcher wasn't able to parse subgroup '%s' into a CollectionRequirement!", subgroups[1])
		return OtherMatcher(group, subgroups)
	}
	return schema.NewChoiceRequirement(collectionReq)
}

// GPAMatcher represents GPA-based prerequisites.
func GPAMatcher(group string, subgroups []string) interface{} {
	GPAFloat, err := strconv.ParseFloat(subgroups[1], 32)
	if err != nil {
		panic(err)
	}
	return schema.NewGPARequirement(GPAFloat, "")
}

// ThrowawayMatcher marks text that should be ignored during requisite evaluation.
func ThrowawayMatcher(group string, subgroups []string) interface{} {
	return schema.Requirement{Type: "throwaway"}
}

// GroupTagMatcher resolves stack-referenced groups by index.
func GroupTagMatcher(group string, subgroups []string) interface{} {
	groupIndex, err := strconv.Atoi(subgroups[1])
	if err != nil {
		panic(err)
	}
	// Return a throwaway if index is out of range
	if groupIndex < 0 || groupIndex >= len(requisiteList) {
		return schema.Requirement{Type: "throwaway"}
	}
	// Find referenced group and return it
	parsedGrp := requisiteList[groupIndex]
	return parsedGrp
}

// OtherMatcher wraps unmatched text in an OtherRequirement.
func OtherMatcher(group string, subgroups []string) interface{} {
	return schema.NewOtherRequirement(ungroupText(group), "")
}

var preOrCoreqRegexp *regexp.Regexp = regexp.MustCompile(`(?i)((?:Prerequisites?\s+or\s+corequisites?|Corequisites?\s+or\s+prerequisites?):(.*))`)
var prereqRegexp *regexp.Regexp = regexp.MustCompile(`(?i)(Prerequisites?:(.*))`)
var coreqRegexp *regexp.Regexp = regexp.MustCompile(`(?i)(Corequisites?:(.*))`)

// Returns a closure that parses the course's requisites
func getReqParser(course *schema.Course, hasEnrollmentReqs bool, enrollmentReqs *goquery.Selection) func() {
	return func() {
		text := course.Description
		if hasEnrollmentReqs {
			course.Enrollment_reqs = utils.TrimWhitespace(enrollmentReqs.Text())
			text = course.Enrollment_reqs
		}

		// Process each section in order, removing matched headers as we go
		sections := []struct {
			regex *regexp.Regexp
			dest  **schema.CollectionRequirement
		}{
			{preOrCoreqRegexp, &course.Co_or_pre_requisites},
			{prereqRegexp, &course.Prerequisites},
			{coreqRegexp, &course.Corequisites},
		}

		for _, sec := range sections {
			match := sec.regex.FindStringSubmatch(text)
			if match == nil {
				continue
			}
			// match[1] is "Prerequisites: ...", match[2] is the inner text
			header, content := match[1], match[2]
			text = strings.Replace(text, header, "", 1) // remove this section from further scanning

			// Remove any other requisite headers nested inside the content
			for _, other := range sections {
				if inner := other.regex.FindStringSubmatch(content); inner != nil {
					content = strings.Replace(content, inner[1], "", -1)
				}
			}

			// Split by ". " and parse each sentence
			var reqs []interface{}
			for _, sentence := range strings.Split(content, ". ") {
				sentence = strings.TrimRight(sentence, ".")
				if req := ParseRequirement(sentence); req != nil && !reqIsThrowaway(req) {
					reqs = append(reqs, req)
				}
			}
			if len(reqs) > 0 {
				*sec.dest = schema.NewCollectionRequirement("REQUISITES", len(reqs), reqs)
			}
		}
	}
}

// This is the list of produced requisites. Indices coincide with group indices -- aka group @0 will also be the 0th index of the list since it will be processed first.
var requisiteList []interface{}

// This is the list of groups that are to be parsed. They are the raw text chunks associated with the reqs above.
var groupList []string

// Function for creating a new group by replacing subtext in an existing group, and pushing the new group's info to the req and group list
func makeSubgroup(group string, subtext string, requisite interface{}) string {
	newGroup := strings.Replace(group, subtext, fmt.Sprintf("@%d", len(requisiteList)), -1)
	requisiteList = append(requisiteList, requisite)
	groupList = append(groupList, newGroup)
	return newGroup
}

// Function for finding the Internal Course Number associated with the course with the specified subject and course number
func findICN(subject string, number string) (string, error) {
	for _, coursePtr := range Courses {
		if coursePtr.Subject_prefix == subject && coursePtr.Course_number == number {
			return coursePtr.Internal_course_number, nil
		}
	}
	return "ERROR", fmt.Errorf("couldn't find an ICN for %s %s", subject, number)
}

// Innermost function for parsing individual text groups (used recursively by some Matchers)
func parseGroup(grp string) interface{} {
	// Make sure we trim any mismatched right parentheses
	grp = strings.TrimRight(grp, ")")
	// Find an applicable matcher in Matchers
	for _, matcher := range Matchers {
		matches := matcher.Regex.FindStringSubmatch(grp)
		if matches != nil {
			// If an applicable matcher has been found, return the result of calling its handler
			result := matcher.Handler(grp, matches)
			utils.VPrintf("'%s' -> %T", grp, result)
			return result
		}
	}
	// If the group couldn't be parsed, give up and make it an OtherRequirement
	utils.VPrintf("'%s' -> parser.OtherRequirement", grp)
	return *schema.NewOtherRequirement(ungroupText(grp), "")
}

// Check whether a requisite is a throwaway or not by trying a type assertion to Requirement
func reqIsThrowaway(req interface{}) bool {
	baseReq, isBaseReq := req.(schema.Requirement)
	return isBaseReq && baseReq.Type == "throwaway"
}

// Use stack-based parentheses parsing to form text groups and reference them in the original string
func groupParens(text string) (string, []string) {
	var groups []string = make([]string, 0, 5)
	var positionStack []int = make([]int, 0, 5)
	var depth int = 0
	for pos := 0; pos < len(text); pos++ {
		if text[pos] == '(' {
			depth++
			positionStack = append(positionStack, pos)
		} else if text[pos] == ')' && depth > 0 {
			depth--
			lastIndex := len(positionStack) - 1
			// Get last '(' position from stack
			lastPos := positionStack[lastIndex]
			// Pop stack
			positionStack = positionStack[:lastIndex]
			// Make group and replace group text with group index reference
			groupText := text[lastPos+1 : pos]
			groupNum := len(groups)
			groups = append(groups, groupText)
			subText := fmt.Sprintf("@%d", groupNum)
			text = strings.Replace(text, text[lastPos:pos+1], subText, -1)
			// Adjust position to account for replaced text
			pos += len(subText) - len(groupText) - 2
		}
	}
	return text, groups
}

// Function for replacing all group references (groups referenced via group tags) with their actual text
func ungroupText(text string) string {
	text = utils.TrimWhitespace(text)
	for groupNum := len(groupList) - 1; groupNum >= 0; groupNum-- {
		subText := fmt.Sprintf("@%d", groupNum)
		replacementText := fmt.Sprintf("(%s)", groupList[groupNum])
		text = strings.Replace(text, subText, replacementText, -1)
	}
	return text
}
