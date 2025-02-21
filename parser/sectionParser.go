package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var sectionPrefixRegexp *regexp.Regexp = utils.Regexpf(`^(?i)%s\.(%s)`, utils.R_SUBJ_COURSE, utils.R_SECTION_CODE)
var coreRegexp *regexp.Regexp = regexp.MustCompile(`[0-9]{3}`)
var personRegexp *regexp.Regexp = regexp.MustCompile(`(.+)・(.+)・(.+)`)

func parseSection(courseRef *schema.Course, classNum string, syllabusURI string, session schema.AcademicSession, rowInfo map[string]*goquery.Selection, classInfo map[string]string) {
	// Get subject prefix and course number by doing a regexp match on the section id
	sectionId := classInfo["Class Section:"]
	idMatches := sectionPrefixRegexp.FindStringSubmatch(sectionId)

	section := &schema.Section{}

	section.Id = primitive.NewObjectID()
	section.Section_number = idMatches[1]
	section.Course_reference = courseRef.Id

	//TODO: section requisites?

	// Set academic session
	section.Academic_session = session
	// Add professors
	section.Professors = parseProfessors(section.Id, rowInfo, classInfo)

	// Get all TA/RA info
	assistantText := utils.TrimWhitespace(rowInfo["TA/RA(s):"].Text())
	assistantMatches := personRegexp.FindAllStringSubmatch(assistantText, -1)
	section.Teaching_assistants = make([]schema.Assistant, 0, len(assistantMatches))
	for _, match := range assistantMatches {
		assistant := schema.Assistant{}
		nameStr := utils.TrimWhitespace(match[1])
		names := strings.Split(nameStr, " ")
		assistant.First_name = strings.Join(names[:len(names)-1], " ")
		assistant.Last_name = names[len(names)-1]
		assistant.Role = utils.TrimWhitespace(match[2])
		assistant.Email = utils.TrimWhitespace(match[3])
		section.Teaching_assistants = append(section.Teaching_assistants, assistant)
	}

	section.Internal_class_number = classNum
	section.Instruction_mode = classInfo["Instruction Mode:"]
	section.Meetings = getMeetings(rowInfo)

	// Parse core flags (may or may not exist)

	if coreText, hasCore := rowInfo["Core:"]; hasCore {
		section.Core_flags = coreRegexp.FindAllString(utils.TrimWhitespace(coreText.Text()), -1)
	}

	section.Syllabus_uri = syllabusURI

	if semesterGrades, ok := GradeMap[session.Name]; ok {
		// We have to trim leading zeroes from the section number in order to match properly, since the grade data does not use leading zeroes
		trimmedSectionNumber := strings.TrimLeft(section.Section_number, "0")
		// Key into grademap should be uppercased like the grade data
		gradeKey := strings.ToUpper(courseRef.Subject_prefix + courseRef.Course_number + trimmedSectionNumber)
		sectionGrades, exists := semesterGrades[gradeKey]
		if exists {
			section.Grade_distribution = sectionGrades
		}
	}

	// Add new section to section map
	Sections[section.Id] = section

	// Append new section to course's section listing
	courseRef.Sections = append(courseRef.Sections, section.Id)
}

func getAcademicSession(rowInfo map[string]*goquery.Selection) schema.AcademicSession {
	session := schema.AcademicSession{}

	for _, node := range rowInfo["Schedule:"].FindMatcher(goquery.Single("p.courseinfo__sectionterm")).Contents().Nodes {
		if node.DataAtom == atom.B {
			//since the key is not a TextElement, the Text is stored in it's first child, a TextElement
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

var meetingDatesRegexp = utils.Regexpf(utils.R_DATE_MDY)
var meetingDaysRegexp = utils.Regexpf(utils.R_WEEKDAY)
var meetingTimesRegexp = utils.Regexpf(utils.R_TIME_AM_PM)

func getMeetings(rowInfo map[string]*goquery.Selection) []schema.Meeting {
	meetingItems := rowInfo["Schedule:"].Find("div.courseinfo__meeting-item--multiple")
	var meetings []schema.Meeting = make([]schema.Meeting, 0, meetingItems.Length())

	meetingItems.Each(func(i int, s *goquery.Selection) {
		meeting := schema.Meeting{}
		meetingInfo := s.FindMatcher(goquery.Single("p.courseinfo__meeting-time"))

		dates := meetingDatesRegexp.FindAllString(meetingInfo.Text(), -1)
		if len(dates) > 0 {
			meeting.Start_date = parseTimeOrPanic(dates[0])

			//There is an edge case where there is only a start date ie "January 2, 2006 (Single Day)"
			if len(dates) == 1 {
				meeting.End_date = meeting.Start_date
			} else {
				meeting.End_date = parseTimeOrPanic(dates[1])
			}
		}

		meetingText := utils.TrimWhitespace(meetingInfo.Contents().FilterFunction(
			func(i int, s *goquery.Selection) bool {
				return s.Nodes[0].Type == html.TextNode
			}).Text())

		//json will convert []string{} into [] rather than null
		if days := meetingDaysRegexp.FindAllString(meetingText, -1); days != nil {
			meeting.Meeting_days = days
		} else {
			meeting.Meeting_days = []string{}
		}

		//Mirroring the handling of start/end date
		times := meetingTimesRegexp.FindAllString(meetingText, -1)
		if len(times) > 0 {
			meeting.Start_time = times[0]

			if len(times) == 1 {
				meeting.End_time = meeting.Start_time
			} else {
				meeting.End_time = times[1]
			}
		}

		if locationInfo := meetingInfo.Find("a"); locationInfo != nil {
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
				} else {
					panic(fmt.Errorf("unable to parse location %s", locationInfo.Text()))
				}
			}
		}
		meetings = append(meetings, meeting)
	})
	return meetings
}

const timeLayout = "January 2, 2006"

func parseTimeOrPanic(value string) time.Time {
	date, err := time.ParseInLocation(timeLayout, value, timeLocation)
	if err != nil {
		panic(err)
	}
	return date
}
