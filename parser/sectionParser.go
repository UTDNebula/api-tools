package parser

import (
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/net/html/atom"
)

// timeLayout is the layout string used for parsing dates in "Month Day, Year" format.
const timeLayout = "January 2, 2006"

var (
	// parser.sectionPrefixRegexp matches "SUBJ.101" (e.g., "HIST.101", case-insensitive).
	sectionPrefixRegexp = utils.Regexpf(`^(?i)%s\.(%s)`, utils.R_SUBJ_COURSE, utils.R_SECTION_CODE)

	// coreRegexp matches any 3-digit number, used for core curriculum codes (e.g., "090").
	coreRegexp = regexp.MustCompile(`[0-9]{3}`)

	// personRegexp matches any 3 strings (no spaces) seperated by '・', (e.g, Name・Role・Email)
	personRegexp = regexp.MustCompile(`(.+)・(.+)・(.+)`)

	// meetingDatesRegexp matches a full date in "Month Day, Year" format (e.g., "January 5, 2022")
	meetingDatesRegexp = regexp.MustCompile(utils.R_DATE_MDY)

	// meetingDaysRegexp matches any day of the week (e.g., Monday, Tuesday, etc.)
	meetingDaysRegexp = regexp.MustCompile(utils.R_WEEKDAY)

	// meetingTimesRegexp matches a time in 12-hour AM/PM format (e.g., "5:00 pm", "11:30am")
	meetingTimesRegexp = regexp.MustCompile(utils.R_TIME_AM_PM)
)

// parseSection creates a schema.Section from rowInfo and ClassInfo,
// adds it to Sections, and updates the associated Course and Professors.
// Internally calls parseCourse and parseProfessors, which modify global maps.
func parseSection(rowInfo map[string]*goquery.Selection, classInfo map[string]string) {
	classNum, courseNum := getInternalClassAndCourseNum(classInfo)
	session := getAcademicSession(rowInfo)
	course := parseCourse(courseNum, session, rowInfo, classInfo)

	id := primitive.NewObjectID()
	sectionNumber := getSectionNumber(classInfo)

	courseRef := schema.CourseRef{
		Prefix: course.Subject_prefix,
		Number: course.Course_number,
		Year:   course.Catalog_year,
	}

	section := schema.Section{
		Id:                    id,
		Section_number:        sectionNumber,
		Course_reference:      &courseRef,
		Academic_session:      session,
		Professors:            parseProfessors(id, sectionNumber, rowInfo, *course, session),
		Teaching_assistants:   getTeachingAssistants(rowInfo),
		Internal_class_number: classNum,
		Instruction_mode:      getInstructionMode(classInfo),
		Meetings:              getMeetings(rowInfo),
		Core_flags:            getCoreFlags(rowInfo),
		Syllabus_uri:          getSyllabusUri(rowInfo),
		Grade_distribution:    getGradeDistribution(session, sectionNumber, course),
	}

	// Add new section to section map
	Sections[courseRef.Prefix+courseRef.Number+sectionNumber+session.Name] = &section

	// Append new section to course's section listing
	course.Sections = append(course.Sections, schema.CourseSectionRef{
		Term:           session.Name,
		Section_number: sectionNumber,
	})
}

// getInternalClassAndCourseNum returns a sections internal course and class number,
// both 0-padded, 5-digit numbers as strings.
// It expects ClassInfo to contain "Class/Course Number:" key.
// If the key is not found or the value is not in the expected "classNum / courseNum" format,
// it returns empty strings.
func getInternalClassAndCourseNum(classInfo map[string]string) (string, string) {
	if numbers, ok := classInfo["Class/Course Number:"]; ok {
		classAndCourseNum := strings.Split(numbers, " / ")
		if len(classAndCourseNum) == 2 {
			return classAndCourseNum[0], classAndCourseNum[1]
		}
		panic("failed to parse internal class number and course number")
	}
	panic("could not find 'Class/Course Number:' in ClassInfo")
}

// getAcademicSession returns the schema.AcademicSession parsed from the provided rowInfo.
// It extracts academic session details from the "Schedule:" section in rowInfo.
func getAcademicSession(rowInfo map[string]*goquery.Selection) schema.AcademicSession {
	session := schema.AcademicSession{}

	infoNodes := rowInfo["Schedule:"].FindMatcher(goquery.Single("p.courseinfo__sectionterm")).Contents().Nodes
	for _, node := range infoNodes {
		if node.DataAtom == atom.B {
			//since the key is not a TextElement, the Text is stored in its first child, a TextElement
			key := utils.TrimWhitespace(node.FirstChild.Data)
			value := utils.TrimWhitespace(node.NextSibling.Data)

			switch key {
			case "Term:":
				session.Name = value
			case "Starts:":
				session.Start_date = parseTimeOrPanic(value)
			case "Ends:":
				session.End_date = parseTimeOrPanic(value)
			}
		}
	}

	if session.Name == "" {
		panic("failed to find academic session, session name can not be empty")
	}

	return session
}

// getSectionNumber returns the matched value from a sectionPrefixRegexp on
// `ClassInfo["Class Section:"]`. It expects ClassInfo to contain "Class Section:" key.
// If there is no matches, getSectionNumber will panic as sectionNumber is a required
// field.
func getSectionNumber(classInfo map[string]string) string {
	if syllabus, ok := classInfo["Class Section:"]; ok {
		matches := sectionPrefixRegexp.FindStringSubmatch(syllabus)
		if len(matches) == 2 {
			return matches[1]
		}
		panic("failed to parse section number")
	}
	panic("could not find 'Class Section:' in ClassInfo")
}

// getTeachingAssistants parses TA/RA information from rowInfo and returns a list of schema.Assistant.
// Assistants are found by matching personRegexp, and therefore are expected to have Name, Role, and Email.
// If no "TA/RA(s):" row is found in rowInfo or no assistants are parsed, an empty slice is returned.
func getTeachingAssistants(rowInfo map[string]*goquery.Selection) []schema.Assistant {
	taRow, ok := rowInfo["TA/RA(s):"]
	if !ok {
		return []schema.Assistant{}
	}
	assistantMatches := personRegexp.FindAllStringSubmatch(utils.TrimWhitespace(taRow.Text()), -1)
	assistants := make([]schema.Assistant, 0, len(assistantMatches))

	for _, match := range assistantMatches {
		names := strings.Split(utils.TrimWhitespace(match[1]), " ")

		assistant := schema.Assistant{
			First_name: strings.Join(names[:len(names)-1], " "),
			Last_name:  names[len(names)-1],
			Role:       utils.TrimWhitespace(match[2]),
			Email:      utils.TrimWhitespace(match[3]),
		}
		assistants = append(assistants, assistant)
	}
	return assistants
}

// getInstructionMode returns the instruction mode (e.g., in-person, online) from ClassInfo.
// It expects ClassInfo to contain "Instruction Mode:" key.
// If the key is not present, it returns an empty string.
func getInstructionMode(classInfo map[string]string) string {
	if mode, ok := classInfo["Instruction Mode:"]; ok {
		return mode
	}
	return ""
}

// getMeetings parses meeting schedule information from the row information map.
//
// The function does not guarantee any number of meetings nor any fields of
// each meeting. Therefore, both an empty slice or a slice containing a meeting
// where all its values are empty are perfectly valid.
//
// Each meeting is parsed as following:
//
// Start and End Date
//   - Accepts 0, 1 or 2 dates matched using meetingDatesRegexp.
//   - If only 1 date is specified, it is used for both dates.
//
// Start and End Time
//   - Accepts 0, 1 or 2 times matched using meetingTimesRegexp.
//   - If only 1 time is specified, it is used for both times.
//   - Times are only parsed into strings to save memory
//
// Meeting days
//   - Captures all strings that match meetingDaysRegexp
//   - If there are no matches an empty slice will be used
//
// Location
//   - Skips locations that don't have a valid locator link
//   - Skips locations whose text don't match format <any><space><any>
func getMeetings(rowInfo map[string]*goquery.Selection) []schema.Meeting {
	meetingItems := rowInfo["Schedule:"].Find("div.courseinfo__meeting-item--multiple")
	var meetings = make([]schema.Meeting, 0, meetingItems.Length())

	meetingItems.Each(func(i int, s *goquery.Selection) {
		meeting := schema.Meeting{}
		meetingInfo := s.FindMatcher(goquery.Single("p.courseinfo__meeting-time"))

		dates := meetingDatesRegexp.FindAllString(meetingInfo.Text(), -1)
		if len(dates) == 2 {
			meeting.Start_date = parseTimeOrPanic(dates[0])
			meeting.End_date = parseTimeOrPanic(dates[1])
		} else if len(dates) == 1 {
			meeting.Start_date = parseTimeOrPanic(dates[0])
			meeting.End_date = meeting.Start_date
		}

		days := meetingDaysRegexp.FindAllString(meetingInfo.Text(), -1)
		if days != nil {
			meeting.Meeting_days = days
		} else {
			meeting.Meeting_days = []string{} //avoid null in the json
		}

		times := meetingTimesRegexp.FindAllString(meetingInfo.Text(), -1)
		if len(times) == 2 {
			meeting.Start_time = times[0]
			meeting.End_time = times[1]
		} else if len(times) == 1 {
			meeting.Start_time = times[0]
			meeting.End_time = meeting.Start_time
		}

		if locationInfo := meetingInfo.FindMatcher(goquery.Single("a")); locationInfo != nil {
			mapUri := locationInfo.AttrOr("href", "")

			//only add locations for meetings that have actual data, all meetings have a link some are not visible or empty
			if mapUri != "" && mapUri != "https://locator.utdallas.edu/" && mapUri != "https://locator.utdallas.edu/ONLINE" {
				splitText := strings.Split(utils.TrimWhitespace(locationInfo.Text()), " ")

				if len(splitText) == 2 {
					meeting.Location = schema.Location{
						Building: splitText[0],
						Room:     splitText[1],
						Map_uri:  mapUri,
					}
				}
			}
		}
		meetings = append(meetings, meeting)
	})
	return meetings
}

// getCoreFlags extracts any matching core curriculum flags from rowInfo.
// It expects rowInfo to contain "Core:" key.
// Core curriculum flags are expected to be 3-digit numbers.
// Returns an empty slice if no "Core:" row is found or no flags are found.
func getCoreFlags(rowInfo map[string]*goquery.Selection) []string {
	if core, ok := rowInfo["Core:"]; ok {
		flags := coreRegexp.FindAllString(utils.TrimWhitespace(core.Text()), -1)

		if flags != nil {
			return flags
		}
	}
	return []string{}
}

// getSyllabusUri extracts and returns the syllabus URL from rowInfo, if present.
// It expects rowInfo to contain "Syllabus:" key, and the syllabus URL to be within an <a> tag.
// Returns an empty string if no "Syllabus:" row or link is found.
func getSyllabusUri(rowInfo map[string]*goquery.Selection) string {
	if syllabus, ok := rowInfo["Syllabus:"]; ok {
		link := syllabus.FindMatcher(goquery.Single("a"))
		if link.Length() == 1 {
			return link.AttrOr("href", "")
		}
	}
	return ""
}

// getGradeDistribution returns the grade distribution for the given section.
// It retrieves grade distribution from the global `GradeMap`.
//
// If GradeMap contains the resulting key it will return the specified slice,
// otherwise it will return an empty slice, `[]int{}`.
// The key is generated using the following formula:
// key = SubjectPrefix + CourseNumber + InternalSectionNumber.
// Note that the InternalSectionNumber is trimmed of leading '0's
func getGradeDistribution(session schema.AcademicSession, sectionNumber string, courseRef *schema.Course) []int {
	if semesterGrades, ok := GradeMap[session.Name]; ok {
		// We have to trim leading zeroes from the section number in order to match properly, since the grade data does not use leading zeroes
		trimmedSectionNumber := strings.TrimLeft(sectionNumber, "0")
		// Key into grademap should be uppercased like the grade data
		gradeKey := strings.ToUpper(courseRef.Subject_prefix + courseRef.Course_number + trimmedSectionNumber)
		sectionGrades, exists := semesterGrades[gradeKey]
		if exists {
			return sectionGrades
		}
	}
	return []int{}
}

// parseTimeOrPanic is a simplified version time.ParseInLocation. The layout and
// location are constants, timeLayout and timeLocation respectively. If time.ParseInLocation
// returns an error, parseTimeOrPanic will panic regardless of the error type.
func parseTimeOrPanic(value string) time.Time {
	date, err := time.ParseInLocation(timeLayout, value, timeLocation)
	if err != nil {
		panic(err)
	}
	return date
}
