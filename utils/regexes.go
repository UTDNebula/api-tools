/*
	This file simply acts as a space to store useful regexp pattern constants for consistency across the project.
*/

package utils

// R_SUBJECT matches a subject prefix such as HIST.
const R_SUBJECT string = `[A-Z]{2,4}`

// R_COURSE_CODE matches a four-character course number like 2252 or V001.
// The first digit of a course code is the course level, the second digit is the number of credit hours.
const R_COURSE_CODE string = `[0-9vV]{4}`

// R_SUBJ_COURSE_CAP captures both subject and course number components.
const R_SUBJ_COURSE_CAP string = `([A-Z]{2,4})\s*([0-9vV]{4})`

// R_SUBJ_COURSE matches subject and course combinations without capturing groups.
const R_SUBJ_COURSE string = `[A-Z]{2,4}\s*[0-9vV]{4}`

// R_SECTION_CODE matches section identifiers such as 101 or A1.
const R_SECTION_CODE string = `[0-9A-z]+`

// R_TERM_CODE matches term codes like 22S or 23f.
const R_TERM_CODE string = `[0-9]{2}[sufSUF]`

// R_GRADE matches letter grades with optional modifiers, such as C-.
const R_GRADE string = `[ABCFabcf][+-]?`

// R_DATE_MDY matches dates formatted like January 5, 2022.
const R_DATE_MDY string = `[A-z]+\s+[0-9]+,\s+[0-9]{4}`

// R_WEEKDAY matches full weekday names like Monday or Thursday.
const R_WEEKDAY string = `(?:Mon|Tues|Wednes|Thurs|Fri|Satur|Sun)day`

// R_TIME_AM_PM matches 12-hour times such as 5:22pm.
const R_TIME_AM_PM string = `[0-9]+:[0-9]+\s*(?:am|pm)`

// R_YEARS matches class standing descriptors like freshmen or seniors.
const R_YEARS string = `(?:freshm[ae]n|sophomores?|juniors?|seniors?)`
