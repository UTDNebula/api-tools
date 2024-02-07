/*
	This file simply acts as a space to store useful regexp pattern constants for consistency across the project.
*/

package utils

// Subject, i.e. HIST
const R_SUBJECT string = `[A-Z]{2,4}`

// Course code, i.e. 2252.
// The first digit of a course code is the course level, the second digit is the # of credit hours.
const R_COURSE_CODE string = `[0-9v]{4}`

// Subject + Course, captured
const R_SUBJ_COURSE_CAP string = `([A-Z]{2,4})\s*([0-9V]{4})`

// Subject + Course, uncaptured
const R_SUBJ_COURSE string = `[A-Z]{2,4}\s*[0-9V]{4}`

// Section code, i.e. 101
const R_SECTION_CODE string = `[0-9A-z]+`

// Term/Semester code, i.e. 22s
const R_TERM_CODE string = `[0-9]{2}[sufSUF]`

// Grade, i.e. C-
const R_GRADE string = `[ABCFabcf][+-]?`

// Date in <MONTH DAY, YEAR> format, i.e. January 5, 2022
const R_DATE_MDY string = `[A-z]+\s+[0-9]+,\s+[0-9]{4}`

// Day of week, i.e. Monday
const R_WEEKDAY string = `(?:Mon|Tues|Wednes|Thurs|Fri|Satur|Sun)day`

// Time in 12-hour AM/PM format, i.e. 5:22pm
const R_TIME_AM_PM string = `[0-9]+:[0-9]+\s*(?:am|pm)`

// Year statuses
const R_YEARS string = `(?:freshm[ae]n|sophomores?|juniors?|seniors?)`
