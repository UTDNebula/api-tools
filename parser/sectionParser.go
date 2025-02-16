package parser

import (
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

	for _, node := range rowInfo["Schedule:"].Find("p.courseinfo__sectionterm").Contents().Nodes {
		if node.DataAtom == atom.B {
			//since the key is not a TextElement, the Text is stored in it's first child, a TextElement
			key := utils.TrimWhitespace(node.FirstChild.Data)
			value := utils.TrimWhitespace(node.NextSibling.Data)

			switch key {
			case "Term:":
				session.Name = value
			case "Starts:":
				if date, err := time.ParseInLocation("January 2, 2006", value, timeLocation); err != nil {
					panic(err)
				} else {
					session.Start_date = date
				}
			case "Ends:":
				if date, err := time.ParseInLocation("January 2, 2006", value, timeLocation); err != nil {
					panic(err)
				} else {
					session.Start_date = date
				}
				//case "Type:" value = "Regular Academic Session"
				//schema.AcademicSession doesn't use type
			}
		}
	}
	return session
}

// separating read the regexes makes it easier to read and allows better handling of edge cases where data is missing
var meetingDatesRegexp = utils.Regexpf(utils.R_DATE_MDY)
var meetingDaysRegexp = utils.Regexpf(utils.R_WEEKDAY)
var meetingTimesRegexp = utils.Regexpf(utils.R_TIME_AM_PM)

func getMeetings(rowInfo map[string]*goquery.Selection) []schema.Meeting {
	var meetings []schema.Meeting = make([]schema.Meeting, 0, 10)

	rowInfo["Schedule:"].Find("div.courseinfo__meeting-item--multiple").Each(func(i int, s *goquery.Selection) {
		meeting := schema.Meeting{}
		meetingInfo := s.Find("p.courseinfo__meeting-time")

		dates := meetingDatesRegexp.FindAllString(meetingInfo.Text(), -1)
		if len(dates) > 0 {
			startDate, err := time.ParseInLocation("January 2, 2006", dates[0], timeLocation)
			if err != nil {
				panic(err)
			}
			meeting.Start_date = startDate

			//There is an edge case where there is only a start date ie "January 2, 2006 (Single Day)"
			if len(dates) == 1 {
				meeting.End_date = meeting.Start_date
			} else {
				endDate, err := time.ParseInLocation("January 2, 2006", dates[1], timeLocation)
				if err != nil {
					panic(err)
				}
				meeting.End_date = endDate
			}
		}

		meetingText := utils.TrimWhitespace(meetingInfo.Contents().FilterFunction(
			func(i int, s *goquery.Selection) bool {
				return s.Nodes[0].Type == html.TextNode
			}).Text())

		meeting.Meeting_days = meetingDaysRegexp.FindAllString(meetingText, -1)
		times := meetingTimesRegexp.FindAllString(meetingText, -1)

		//len checks are necessary since some meetings don't include times or have something like tbd-tbd
		if len(times) > 0 {
			meeting.Start_time = times[0]
			if len(times) > 1 {
				meeting.End_time = times[1]
			} else {
				meeting.Start_time = times[0]
			}
		}

		//only adding locations for meetings that have actual data, all meetings have a link some are not visible and empty
		if locationInfo := meetingInfo.Find("a"); locationInfo != nil {
			map_uri := locationInfo.AttrOr("href", "")

			//don't include location for remote classes or classes without locations
			//map uri should never be "" but doesn't hurt to check
			if map_uri != "" && map_uri != "https://locator.utdallas.edu/" && map_uri != "https://locator.utdallas.edu/ONLINE" {
				splitText := strings.Split(strings.TrimSpace(locationInfo.Text()), " ")

				if len(splitText) == 1 {
					panic("")
				}
				meeting.Location = schema.Location{
					Building: splitText[0],
					Room:     splitText[1],
					Map_uri:  locationInfo.AttrOr("href", ""),
				}
			}

		}
		meetings = append(meetings, meeting)
	})
	return meetings
}
