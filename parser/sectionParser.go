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

const timeLayout = "January 2, 2006"

var (
	// sectionPrefixRegexp the regular expression used to
	sectionPrefixRegexp = utils.Regexpf(`^(?i)%s\.(%s)`, utils.R_SUBJ_COURSE, utils.R_SECTION_CODE)
	coreRegexp          = regexp.MustCompile(`[0-9]{3}`)
	personRegexp        = regexp.MustCompile(`(.+)・(.+)・(.+)`)
	meetingDatesRegexp  = regexp.MustCompile(utils.R_DATE_MDY)
	meetingDaysRegexp   = regexp.MustCompile(utils.R_WEEKDAY)
	meetingTimesRegexp  = regexp.MustCompile(utils.R_TIME_AM_PM)
)

// parseSection
func parseSection(rowInfo map[string]*goquery.Selection, classInfo map[string]string) {
	classNum, courseNum := getInternalClassAndCourseNum(classInfo)
	session := getAcademicSession(rowInfo)
	courseRef := parseCourse(courseNum, session, rowInfo, classInfo)

	sectionNumber := getSectionNumber(classInfo)

	id := primitive.NewObjectID()

	section := schema.Section{
		Id:                    id,
		Section_number:        sectionNumber,
		Course_reference:      courseRef.Id,
		Academic_session:      session,
		Professors:            parseProfessors(id, rowInfo, classInfo),
		Teaching_assistants:   getTeachingAssistants(rowInfo),
		Internal_class_number: classNum,
		Instruction_mode:      getInstructionMode(classInfo),
		Meetings:              getMeetings(rowInfo),
		Core_flags:            getCoreFlags(rowInfo),
		Syllabus_uri:          getSyllabusUri(rowInfo),
		Grade_distribution:    getGradeDistribution(session, sectionNumber, courseRef),
	}

	// Add new section to section map
	Sections[section.Id] = &section

	// Append new section to course's section listing
	courseRef.Sections = append(courseRef.Sections, section.Id)
}

// getInternalClassAndCourseNum returns a sections internal course and class number,
// both 0-padded, 5-digit numbers.
//
// Found in a sections `Class Info` table under `Class/Course Number:`
func getInternalClassAndCourseNum(classInfo map[string]string) (string, string) {
	if numbers, ok := classInfo["Class/Course Number:"]; ok {
		classAndCourseNum := strings.Split(numbers, " / ")
		if len(classAndCourseNum) == 2 {
			return classAndCourseNum[0], classAndCourseNum[1]
		}
	}
	return "", ""
}

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
	return session
}

func getSectionNumber(classInfo map[string]string) string {
	if syllabus, ok := classInfo["Class Section:"]; ok {
		matches := sectionPrefixRegexp.FindStringSubmatch(syllabus)
		if len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

func getTeachingAssistants(rowInfo map[string]*goquery.Selection) []schema.Assistant {
	assistantMatches := personRegexp.FindAllStringSubmatch(utils.TrimWhitespace(rowInfo["TA/RA(s):"].Text()), -1)
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
func getMeetings(rowInfo map[string]*goquery.Selection) []schema.Meeting {
	meetingItems := rowInfo["Schedule:"].Find("div.courseinfo__meeting-item--multiple")
	var meetings []schema.Meeting = make([]schema.Meeting, 0, meetingItems.Length())

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

func getCoreFlags(rowInfo map[string]*goquery.Selection) []string {
	if core, ok := rowInfo["Core:"]; ok {
		flags := coreRegexp.FindAllString(utils.TrimWhitespace(core.Text()), -1)

		if flags != nil {
			return flags
		}
	}
	return []string{}
}

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
//
// If GradeMap contains the resulting key it will return the specified slice,
// otherwise it will return an empty slice, `[]int{}`.
// The key is generated using the following formula:
// key = SubjectPrefix + InternalCourseNumber + InternalSectionNumber.
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
// returns an error, parseTimeOrPanic will panic regardless of the error type or reason.
func parseTimeOrPanic(value string) time.Time {
	date, err := time.ParseInLocation(timeLayout, value, timeLocation)
	if err != nil {
		panic(err)
	}
	return date
}
