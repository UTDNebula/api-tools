package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var sectionPrefixRegexp *regexp.Regexp = utils.Regexpf(`^(?i)%s\.(%s)`, utils.R_SUBJ_COURSE, utils.R_SECTION_CODE)
var coreRegexp *regexp.Regexp = regexp.MustCompile(`[0-9]{3}`)
var personRegexp *regexp.Regexp = regexp.MustCompile(`(.+)・(.+)・(.+)`)

func parseSection(courseRef *schema.Course, classNum string, syllabusURI string, session schema.AcademicSession, rowInfo map[string]string, classInfo map[string]string) {
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
	assistantText := rowInfo["TA/RA(s):"]
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
	section.Meetings = getMeetings(rowInfo, classInfo)

	// Parse core flags (may or may not exist)
	coreText, hasCore := rowInfo["Core:"]
	if hasCore {
		section.Core_flags = coreRegexp.FindAllString(coreText, -1)
	}

	section.Syllabus_uri = syllabusURI

	semesterGrades, exists := GradeMap[session.Name]
	if exists {
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

var termRegexp *regexp.Regexp = utils.Regexpf(`(?i)Term: (%s)`, utils.R_TERM_CODE)
var datesRegexp *regexp.Regexp = utils.Regexpf(`(?:Start|End)s: (%s)`, utils.R_DATE_MDY)

func getAcademicSession(rowInfo map[string]string) schema.AcademicSession {
	session := schema.AcademicSession{}
	scheduleText := rowInfo["Schedule:"]

	session.Name = termRegexp.FindStringSubmatch(scheduleText)[1]
	dateMatches := datesRegexp.FindAllStringSubmatch(scheduleText, -1)

	datesFound := len(dateMatches)
	switch {
	case datesFound == 1:
		startDate, err := time.ParseInLocation("January 2, 2006", dateMatches[0][1], timeLocation)
		if err != nil {
			panic(err)
		}
		session.Start_date = startDate
	case datesFound == 2:
		startDate, err := time.ParseInLocation("January 2, 2006", dateMatches[0][1], timeLocation)
		if err != nil {
			panic(err)
		}
		endDate, err := time.ParseInLocation("January 2, 2006", dateMatches[1][1], timeLocation)
		if err != nil {
			panic(err)
		}
		session.Start_date = startDate
		session.End_date = endDate
	}
	return session
}

var meetingsRegexp *regexp.Regexp = utils.Regexpf(`(%s)-(%s)\W+((?:%s(?:, )?)+)\W+(%s)-(%s)(?:\W+(?:(\S+)\s+(\S+)))`, utils.R_DATE_MDY, utils.R_DATE_MDY, utils.R_WEEKDAY, utils.R_TIME_AM_PM, utils.R_TIME_AM_PM)

func getMeetings(rowInfo map[string]string, classInfo map[string]string) []schema.Meeting {
	scheduleText := rowInfo["Schedule:"]
	meetingMatches := meetingsRegexp.FindAllStringSubmatch(scheduleText, -1)
	var meetings []schema.Meeting = make([]schema.Meeting, 0, len(meetingMatches))
	for _, match := range meetingMatches {
		meeting := schema.Meeting{}

		startDate, err := time.ParseInLocation("January 2, 2006", match[1], timeLocation)
		if err != nil {
			panic(err)
		}
		meeting.Start_date = startDate

		endDate, err := time.ParseInLocation("January 2, 2006", match[2], timeLocation)
		if err != nil {
			panic(err)
		}
		meeting.End_date = endDate

		meeting.Meeting_days = strings.Split(match[3], ", ")

		// Don't parse time into time object, adds unnecessary extra data
		meeting.Start_time = match[4]
		meeting.End_time = match[5]

		// Only add location data if it's available
		if len(match) > 6 {
			location := schema.Location{}
			location.Building = match[6]
			location.Room = match[7]
			location.Map_uri = fmt.Sprintf("https://locator.utdallas.edu/%s_%s", location.Building, location.Room)
			meeting.Location = location
		}

		meetings = append(meetings, meeting)
	}
	return meetings
}
