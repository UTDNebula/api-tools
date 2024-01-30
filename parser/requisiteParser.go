package parser

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

/*
	Below is the code for the requisite parser. It is *by far* the most complicated code in this entire project.
	In summary, it uses a bottom-up "stack"-based parsing technique, building requisites by taking small groups of text, parsing those groups,
	storing them on the "stack", and then uses those previously parsed groups as dependencies for parsing the larger "higher level" groups.

	It's worth noting that I say stack in quotes above because it's not treated as strictly LIFO like a stack would normally be.
*/

// Regex matcher object for requisite group parsing
type Matcher struct {
	Regex   *regexp.Regexp
	Handler func(string, []string) interface{}
}

////////////////////// BEGIN MATCHER FUNCS //////////////////////

var ANDRegex = regexp.MustCompile(`(?i)\s+and\s+`)

func ANDMatcher(group string, subgroups []string) interface{} {
	// Split text along " and " boundaries, then parse subexpressions as groups into an "AND" CollectionRequirement
	subExpressions := ANDRegex.Split(group, -1)
	parsedSubExps := make([]interface{}, 0, len(subExpressions))
	for _, exp := range subExpressions {
		parsedExp := parseGroup(utils.TrimWhitespace(exp))
		// Don't include throwaways
		if !reqIsThrowaway(parsedExp) {
			parsedSubExps = append(parsedSubExps, parsedExp)
		}
	}

	parsedSubExps = joinAdjacentOthers(parsedSubExps, " and ")

	if len(parsedSubExps) > 1 {
		return schema.NewCollectionRequirement("AND", len(parsedSubExps), parsedSubExps)
	} else {
		return parsedSubExps[0]
	}
}

// First regex subgroup represents the text to be subgrouped and parsed with parseFnc
// Ex: Text is: "(OPRE 3360 or STAT 3360 or STAT 4351), and JSOM majors and minors only"
// Regex is: "(JSOM majors and minors only)"
// Resulting substituted text would be: "(OPRE 3360 or STAT 3360 or STAT 4351), and @N", where N is some group number
// When @N is dereferenced from the requisite list, it will have a value equivalent to the result of parseFnc(group, subgroups)

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

var ORRegex = regexp.MustCompile(`(?i)\s+or\s+`)

func ORMatcher(group string, subgroups []string) interface{} {
	// Split text along " or " boundaries, then parse subexpressions as groups into an "OR" CollectionRequirement
	subExpressions := ORRegex.Split(group, -1)
	parsedSubExps := make([]interface{}, 0, len(subExpressions))
	for _, exp := range subExpressions {
		parsedExp := parseGroup(utils.TrimWhitespace(exp))
		// Don't include throwaways
		if !reqIsThrowaway(parsedExp) {
			parsedSubExps = append(parsedSubExps, parsedExp)
		}
	}

	parsedSubExps = joinAdjacentOthers(parsedSubExps, " or ")

	if len(parsedSubExps) > 1 {
		return schema.NewCollectionRequirement("OR", 1, parsedSubExps)
	} else {
		return parsedSubExps[0]
	}
}

func CourseMinGradeMatcher(group string, subgroups []string) interface{} {
	icn, err := findICN(subgroups[1], subgroups[2])
	if err != nil {
		log.Printf("WARN: %s\n", err)
		return OtherMatcher(group, subgroups)
	}
	return schema.NewCourseRequirement(icn, subgroups[3])
}

func CourseMatcher(group string, subgroups []string) interface{} {
	icn, err := findICN(subgroups[1], subgroups[2])
	if err != nil {
		log.Printf("WARN: %s\n", err)
		return OtherMatcher(group, subgroups)
	}
	return schema.NewCourseRequirement(icn, "D")
}

func ConsentMatcher(group string, subgroups []string) interface{} {
	return schema.NewConsentRequirement(subgroups[1])
}

func LimitMatcher(group string, subgroups []string) interface{} {
	hourLimit, err := strconv.Atoi(subgroups[1])
	if err != nil {
		panic(err)
	}
	return schema.NewLimitRequirement(hourLimit)
}

func MajorMatcher(group string, subgroups []string) interface{} {
	return schema.NewMajorRequirement(subgroups[1])
}

func MinorMatcher(group string, subgroups []string) interface{} {
	return schema.NewMinorRequirement(subgroups[1])
}

func MajorMinorMatcher(group string, subgroups []string) interface{} {
	return schema.NewCollectionRequirement("OR", 1, []interface{}{*schema.NewMajorRequirement(subgroups[1]), *schema.NewMinorRequirement(subgroups[1])})
}

func CoreMatcher(group string, subgroups []string) interface{} {
	hourReq, err := strconv.Atoi(subgroups[1])
	if err != nil {
		panic(err)
	}
	return schema.NewCoreRequirement(subgroups[2], hourReq)
}

func CoreCompletionMatcher(group string, subgroups []string) interface{} {
	return schema.NewCoreRequirement(subgroups[1], -1)
}

func ChoiceMatcher(group string, subgroups []string) interface{} {
	collectionReq, ok := parseGroup(subgroups[1]).(*schema.CollectionRequirement)
	if !ok {
		log.Printf("WARN: ChoiceMatcher wasn't able to parse subgroup '%s' into a CollectionRequirement!", subgroups[1])
		return OtherMatcher(group, subgroups)
	}
	return schema.NewChoiceRequirement(collectionReq)
}

func GPAMatcher(group string, subgroups []string) interface{} {
	GPAFloat, err := strconv.ParseFloat(subgroups[1], 32)
	if err != nil {
		panic(err)
	}
	return schema.NewGPARequirement(GPAFloat, "")
}

func ThrowawayMatcher(group string, subgroups []string) interface{} {
	return schema.Requirement{Type: "throwaway"}
}

// Regex for group tags
var groupTagRegex = regexp.MustCompile(`@(\d+)`)

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

func OtherMatcher(group string, subgroups []string) interface{} {
	return schema.NewOtherRequirement(ungroupText(group), "")
}

/////////////////////// END MATCHER FUNCS ///////////////////////

// Matcher container, matchers must be in order of precedence
// NOTE: PARENTHESES ARE OF HIGHEST PRECEDENCE! (This is due to groupParens() handling grouping of parenthesized text before parsing begins)
var Matchers []Matcher

// Must init matchers via function at runtime to avoid compile-time circular definition error
func initMatchers() {
	Matchers = []Matcher{

		// Throwaways
		{
			regexp.MustCompile(`^(?i)(?:better|\d-\d|same as.+)$`),
			ThrowawayMatcher,
		},

		/* TO IMPLEMENT:

		X or Y or ... Z Major/Minor

		SUBJECT NUMBER, SUBJECT NUMBER, ..., or SUBJECT NUMBER

		... probably many more

		*/

		// * <YEAR> only
		{
			utils.Regexpf(`(?i).+%s\s+only$`, utils.R_YEARS),
			OtherMatcher,
		},

		// * in any combination of *
		{
			regexp.MustCompile(`(?i).+\s+in\s+any\s+combination\s+of\s+.+`),
			OtherMatcher,
		},

		// <SUBJECT> majors and minors only
		{
			utils.Regexpf(`(?i)((%s)\s+majors\s+and\s+minors\s+only)`, utils.R_SUBJECT),
			SubstitutionMatcher(func(group string, subgroups []string) interface{} {
				return MajorMinorMatcher(subgroups[1], subgroups[1:3])
			}),
		},

		// Completion of [a/an] <CORE CODE> core [course]
		{
			regexp.MustCompile(`(?i)(Completion\s+of\s+(?:an?\s+)?(\d{3}).+core(?:\s+course)?)`),
			SubstitutionMatcher(func(group string, subgroups []string) interface{} {
				return CoreCompletionMatcher(subgroups[1], subgroups[1:3])
			}),
		},

		// Credit cannot be received for both [courses][,] <EXPRESSION>
		{
			regexp.MustCompile(`(?i)(Credit\s+cannot\s+be\s+received\s+for\s+both\s+(?:courses)?,?(.+))`),
			SubstitutionMatcher(func(group string, subgroups []string) interface{} {
				return ChoiceMatcher(subgroups[1], subgroups[1:3])
			}),
		},

		// Credit cannot be received for more than one of *: <EXPRESSION>
		{
			regexp.MustCompile(`(?i)(Credit\s+cannot\s+be\s+received\s+for\s+more\s+than\s+one\s+of.+:(.+))`),
			SubstitutionMatcher(func(group string, subgroups []string) interface{} {
				return ChoiceMatcher(subgroups[1], subgroups[1:3])
			}),
		},

		// Logical &
		{
			ANDRegex,
			ANDMatcher,
		},

		// "<COURSE> with a [grade] [of] <GRADE> or better"
		{
			utils.Regexpf(`^(?i)(%s\s+with\s+a(?:\s+grade)?(?:\s+of)?\s+(%s)\s+or\s+better)`, utils.R_SUBJ_COURSE_CAP, utils.R_GRADE), // [name, number, min grade]
			SubstitutionMatcher(func(group string, subgroups []string) interface{} {
				return CourseMinGradeMatcher(subgroups[1], subgroups[1:5])
			}),
		},

		// Logical |
		{
			ORRegex,
			ORMatcher,
		},

		// <COURSE> with a [minimum] grade of [at least] [a] <GRADE>
		{
			utils.Regexpf(`^(?i)%s\s+with\s+a\s+(?:minimum\s+)?grade\s+of\s+(?:at least\s+)?(?:a\s+)?(%s)$`, utils.R_SUBJ_COURSE_CAP, utils.R_GRADE), // [name, number, min grade]
			CourseMinGradeMatcher,
		},

		// A grade of [at least] [a] <GRADE> in <COURSE>
		{
			utils.Regexpf(`^(?i)A\s+grade\s+of(?:\s+at\s+least)?(?:\s+a)?\s+(%s)\s+in\s+%s$`, utils.R_GRADE, utils.R_SUBJ_COURSE_CAP), // [min grade, name, number]
			func(group string, subgroups []string) interface{} {
				return CourseMinGradeMatcher(group, []string{subgroups[0], subgroups[2], subgroups[3], subgroups[1]})
			},
		},

		// <COURSE>
		{
			utils.Regexpf(`^\s*%s\s*$`, utils.R_SUBJ_COURSE_CAP), // [name, number]
			CourseMatcher,
		},

		// <GRANTER> consent required
		{
			regexp.MustCompile(`^(?i)(.+)\s+consent\s+required`), // [granter]
			ConsentMatcher,
		},

		// <HOURS> semester credit hours maximum
		{
			regexp.MustCompile(`^(?i)(\d+)\s+semester\s+credit\s+hours\s+maximum$`),
			LimitMatcher,
		},

		// This course may only be repeated for <HOURS> credit hours
		{
			utils.Regexpf(`^(?:%s\s+)?Repeat\s+Limit\s+-\s+(?:%s|This\s+course)\s+may\s+only\s+be\s+repeated\s+for(?:\s+a\s+maximum\s+of)?\s+(\d+)\s+semester\s+cre?dit\s+hours(?:\s+maximum)?$`, utils.R_SUBJ_COURSE, utils.R_SUBJ_COURSE),
			LimitMatcher,
		},

		// <SUBJECT> majors only
		{
			regexp.MustCompile(`^(?i)(.+)\s+major(?:s\s+only)?$`),
			MajorMatcher,
		},

		// <SUBJECT> minors only
		{
			regexp.MustCompile(`^(?i)(.+)\s+minor(?:s\s+only)?$`),
			MinorMatcher,
		},

		// Any <HOURS> semester credit hour <CORE> course
		{
			regexp.MustCompile(`^(?i)any\s+(\d+)\s+semester\s+credit\s+hour\s+(\d{3})(?:\s+@\d+)?\s+core(?:\s+course)?$`),
			CoreMatcher,
		},

		// Minimum GPA of <GPA>
		{
			regexp.MustCompile(`^(?i)(?:minimum\s+)?GPA\s+of\s+([0-9\.]+)$`), // [GPA]
			GPAMatcher,
		},

		// <GPA> GPA
		{
			regexp.MustCompile(`^(?i)([0-9\.]+) GPA$`), // [GPA]
			GPAMatcher,
		},

		// A university grade point average of at least <GPA>
		{
			regexp.MustCompile(`^(?i)a(?:\s+university)?\s+grade\s+point\s+average\s+of(?:\s+at\s+least)?\s+([0-9\.]+)$`), // [GPA]
			GPAMatcher,
		},

		// Group tags (i.e. @1)
		{
			groupTagRegex, // [group #]
			GroupTagMatcher,
		},
	}
}

var preOrCoreqRegexp *regexp.Regexp = regexp.MustCompile(`(?i)((?:Prerequisites?\s+or\s+corequisites?|Corequisites?\s+or\s+prerequisites?):(.*))`)
var prereqRegexp *regexp.Regexp = regexp.MustCompile(`(?i)(Prerequisites?:(.*))`)
var coreqRegexp *regexp.Regexp = regexp.MustCompile(`(?i)(Corequisites?:(.*))`)

// It is very important that these remain in the same order -- this keeps proper precedence in the below function!
var reqRegexes [3]*regexp.Regexp = [3]*regexp.Regexp{preOrCoreqRegexp, prereqRegexp, coreqRegexp}

// Returns a closure that parses the course's requisites
func getReqParser(course *schema.Course, hasEnrollmentReqs bool, enrollmentReqs string) func() {
	return func() {
		// Pointer array to course requisite properties must be in same order as reqRegexes above
		courseReqs := [3]**schema.CollectionRequirement{&course.Co_or_pre_requisites, &course.Prerequisites, &course.Corequisites}
		// The actual text to check for requisites
		var checkText string
		// Extract req text from the enrollment req info if it exists, otherwise try using the description
		if hasEnrollmentReqs {
			course.Enrollment_reqs = enrollmentReqs
			checkText = enrollmentReqs
		} else {
			checkText = course.Description
		}
		// Iterate over and parse each type of requisite, populating the course's relevant requisite property
		for index, reqPtr := range courseReqs {
			reqMatches := reqRegexes[index].FindStringSubmatch(checkText)
			if reqMatches != nil {
				// Actual useful text is the inner match, index 2
				reqText := reqMatches[2]
				// Erase any sub-matches for other requisite types by matching outer text, index 1
				for _, regex := range reqRegexes {
					matches := regex.FindStringSubmatch(reqText)
					if matches != nil {
						reqText = strings.Replace(reqText, matches[1], "", -1)
					}
				}
				// Erase current match from checkText to prevent erroneous duplicated Reqs
				checkText = strings.Replace(checkText, reqMatches[1], "", -1)
				// Split reqText into chunks based on period-space delimiters
				textChunks := strings.Split(utils.TrimWhitespace(reqText), ". ")
				parsedChunks := make([]interface{}, 0, len(textChunks))
				// Parse each chunk, then add non-throwaway chunks to parsedChunks
				for _, chunk := range textChunks {
					// Trim any remaining rightmost periods
					chunk = utils.TrimWhitespace(strings.TrimRight(chunk, "."))
					parsedChunk := parseChunk(chunk)
					if !reqIsThrowaway(parsedChunk) {
						parsedChunks = append(parsedChunks, parsedChunk)
					}
				}
				// Build CollectionRequirement from parsed chunks and apply to the course property
				if len(parsedChunks) > 0 {
					*reqPtr = schema.NewCollectionRequirement("REQUISITES", len(parsedChunks), parsedChunks)
				}
				log.Printf("\n\n")
			}
		}
	}
}

// Function for pulling all requisite references (reqs referenced via group tags) from text
/*
func getReqRefs(text string) []interface{} {
	matches := groupTagRegex.FindAllStringSubmatch(text, -1)
	refs := make([]interface{}, len(matches))
	for i, submatches := range matches {
		refs[i] = GroupTagMatcher(submatches[0], submatches)
	}
	return refs
}
*/

// Function for creating a new group by replacing subtext in an existing group, and pushing the new group's info to the req and group list
func makeSubgroup(group string, subtext string, requisite interface{}) string {
	newGroup := strings.Replace(group, subtext, fmt.Sprintf("@%d", len(requisiteList)), -1)
	requisiteList = append(requisiteList, requisite)
	groupList = append(groupList, newGroup)
	return newGroup
}

// Function for joining adjacent OtherRequirements into one OtherRequirement by joining their descriptions with a string
func joinAdjacentOthers(reqs []interface{}, joinString string) []interface{} {
	joinedReqs := make([]interface{}, 0, len(reqs))
	// Temp is a blank OtherRequirement
	temp := *schema.NewOtherRequirement("", "")
	// Iterate over each existing req
	for _, req := range reqs {
		// Determine whether req is an OtherRequirement
		otherReq, isOtherReq := req.(schema.OtherRequirement)
		if !isOtherReq {
			// If temp contains data, append its final result to the joinedReqs
			if temp.Description != "" {
				joinedReqs = append(joinedReqs, temp)
			}
			// Append the non-OtherRequirement to the joinedReqs
			joinedReqs = append(joinedReqs, req)
			// Reset temp's description
			temp.Description = ""
			continue
		}
		// If temp is blank, and req is an otherReq, use otherReq as the initial value of temp
		// Otherwise, join temp's existing description with otherReq's description
		if temp.Description == "" {
			temp = otherReq
		} else {
			temp.Description = strings.Join([]string{temp.Description, otherReq.Description}, joinString)
		}
	}
	// If temp contains data, append its final result to the joinedReqs
	if temp.Description != "" {
		joinedReqs = append(joinedReqs, temp)
	}
	//log.Printf("JOINEDREQS ARE: %v\n", joinedReqs)
	return joinedReqs
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

// This is the list of produced requisites. Indices coincide with group indices -- aka group @0 will also be the 0th index of the list since it will be processed first.
var requisiteList []interface{}

// This is the list of groups that are to be parsed. They are the raw text chunks associated with the reqs above.
var groupList []string

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
			log.Printf("'%s' -> %T\n", grp, result)
			return result
		}
	}
	// Panic if no matcher was able to be found for a given group -- this means we need to add handling for it!!!
	//log.Panicf("NO MATCHER FOUND FOR GROUP '%s'\nSTACK IS: %#v\n", grp, requisiteList)
	//log.Printf("NO MATCHER FOR: '%s'\n", grp)
	log.Printf("'%s' -> parser.OtherRequirement\n", grp)
	//var temp string
	//fmt.Scanf("%s", temp)
	return *schema.NewOtherRequirement(ungroupText(grp), "")
}

// Outermost function for parsing a chunk of requisite text (potentially containing multiple nested text groups)
func parseChunk(chunk string) interface{} {
	log.Printf("\nPARSING CHUNK: '%s'\n", chunk)
	// Extract parenthesized groups from chunk text
	parseText, parseGroups := groupParens(chunk)
	// Initialize the requisite list and group list
	requisiteList = make([]interface{}, 0, len(parseGroups))
	groupList = parseGroups
	// Begin recursive group parsing -- order is bottom-up
	for _, grp := range parseGroups {
		parsedReq := parseGroup(grp)
		// Only append requisite to stack if it isn't marked as throwaway
		if !reqIsThrowaway(parsedReq) {
			requisiteList = append(requisiteList, parsedReq)
		}
	}
	finalGroup := parseGroup(parseText)
	return finalGroup
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
