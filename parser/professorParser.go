package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	apiPrimaryLocationRegex  = regexp.MustCompile(`^(\w+)\s+(\d+\.\d{3}[A-Za-z]?)$`)
	apiFallbackLocationRegex = regexp.MustCompile(`^([A-Za-z]+)(\d+)\.?([\d]{3}[A-Za-z]?)$`)
)

func newProfessor(firstName, lastName string) *schema.Professor {
	prof := &schema.Professor{}
	prof.Id = primitive.NewObjectID()
	prof.First_name = firstName
	prof.Last_name = lastName
	return prof
}

func buildProfessorFromRow(row profileIndexRow) *schema.Professor {
	firstName := strings.TrimSpace(row.FirstName)
	lastName := strings.TrimSpace(row.LastName)
	if firstName == "" || lastName == "" {
		firstName, lastName = splitFullName(row.FullName)
	}

	// Ignore blank names to match the parser's existing professor population behavior.
	if firstName == "" || lastName == "" {
		return nil
	}

	prof := newProfessor(firstName, lastName)
	applyProfileFields(prof, row)

	return prof
}

func applyProfileFields(prof *schema.Professor, row profileIndexRow) {
	titles := collectTitles(row)
	info := bestInformationData(row.Information)
	if prof.Titles == nil {
		prof.Titles = []string{}
	}

	for _, title := range titles {
		if !containsString(prof.Titles, title) {
			prof.Titles = append(prof.Titles, title)
		}
	}

	if prof.Email == "" {
		prof.Email = trimNullableString(info.Email)
	}
	if prof.Phone_number == "" {
		prof.Phone_number = trimNullableString(info.Phone)
	}
	if prof.Office.Building == "" && prof.Office.Room == "" && prof.Office.Map_uri == "" {
		prof.Office = bestLocation(row.Information)
	}
	if prof.Profile_uri == "" {
		prof.Profile_uri = bestProfileURI(row)
	}
	if prof.Image_uri == "" {
		prof.Image_uri = bestImageURI(row)
	}
	if prof.Office_hours == nil {
		prof.Office_hours = []schema.Meeting{}
	}
	if prof.Sections == nil {
		prof.Sections = []primitive.ObjectID{}
	}
}

func parseProfessors(sectionId primitive.ObjectID, rowInfo map[string]*goquery.Selection) []primitive.ObjectID {
	professorText := utils.TrimWhitespace(rowInfo["Instructor(s):"].Text())
	professorMatches := personRegexp.FindAllStringSubmatch(professorText, -1)
	var profRefs []primitive.ObjectID = make([]primitive.ObjectID, 0, len(professorMatches))
	for _, match := range professorMatches {

		nameStr := utils.TrimWhitespace(match[1])
		names := strings.Split(nameStr, " ")

		firstName := strings.Join(names[:len(names)-1], " ")
		lastName := names[len(names)-1]

		// Ignore blank names, because they exist for some reason???
		if firstName == "" || lastName == "" {
			continue
		}

		profKey := firstName + lastName
		email := utils.TrimWhitespace(match[3])

		prof, profExists := Professors[profKey]
		if profExists {
			prof.Sections = append(prof.Sections, sectionId)
			if prof.Email == "" {
				prof.Email = email
			}
			profRefs = append(profRefs, prof.Id)
			continue
		}

		if profByEmail, emailMatch := findProfessorByEmail(email); emailMatch {
			profByEmail.Sections = append(profByEmail.Sections, sectionId)
			if profByEmail.Email == "" {
				profByEmail.Email = email
			}
			if _, exists := ProfessorIDMap[profByEmail.Id]; !exists {
				ProfessorIDMap[profByEmail.Id] = profKey
			}
			Professors[profKey] = profByEmail
			profRefs = append(profRefs, profByEmail.Id)
			continue
		}

		prof = newProfessor(firstName, lastName)
		prof.Titles = []string{utils.TrimWhitespace(match[2])}
		prof.Email = email
		prof.Sections = []primitive.ObjectID{sectionId}
		profRefs = append(profRefs, prof.Id)
		Professors[profKey] = prof
		ProfessorIDMap[prof.Id] = profKey
	}
	return profRefs
}

func findProfessorByEmail(email string) (*schema.Professor, bool) {
	normalized := strings.TrimSpace(strings.ToLower(email))
	if normalized == "" {
		return nil, false
	}

	for _, prof := range Professors {
		if prof == nil {
			continue
		}
		if strings.TrimSpace(strings.ToLower(prof.Email)) == normalized {
			return prof, true
		}
	}

	return nil, false
}

func splitFullName(fullName string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(fullName))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return strings.Join(parts[:len(parts)-1], " "), parts[len(parts)-1]
}

func parseAPILocation(text string) schema.Location {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return schema.Location{}
	}

	var building string
	var room string

	submatches := apiPrimaryLocationRegex.FindStringSubmatch(normalized)
	if submatches == nil {
		submatches = apiFallbackLocationRegex.FindStringSubmatch(strings.ReplaceAll(normalized, " ", ""))
		if submatches == nil {
			return schema.Location{}
		}
		building = submatches[1]
		room = fmt.Sprintf("%s.%s", submatches[2], submatches[3])
	} else {
		building = submatches[1]
		room = submatches[2]
	}

	return schema.Location{
		Building: building,
		Room:     room,
		Map_uri:  fmt.Sprintf("https://locator.utdallas.edu/%s_%s", building, room),
	}
}

func collectTitles(row profileIndexRow) []string {
	titles := make([]string, 0, 8)
	if row.Name != "" {
		titles = append(titles, strings.TrimSpace(row.Name))
	}

	for _, info := range row.Information {
		for _, candidate := range []*string{info.Data.Title, info.Data.SecondaryTitle, info.Data.TertiaryTitle, info.Data.DistinguishedTitle} {
			trimmed := trimNullableString(candidate)
			if trimmed == "" {
				continue
			}
			if !containsString(titles, trimmed) {
				titles = append(titles, trimmed)
			}
		}
	}

	return titles
}

func bestInformationData(items []profileInformation) profileInformationData {
	if len(items) == 0 {
		return profileInformationData{}
	}

	best := items[0].Data
	bestScore := informationScore(best)

	for _, item := range items[1:] {
		score := informationScore(item.Data)
		if score > bestScore {
			best = item.Data
			bestScore = score
		}
	}

	return best
}

func informationScore(data profileInformationData) int {
	score := 0
	for _, value := range []string{
		trimNullableString(data.Email),
		trimNullableString(data.Phone),
		trimNullableString(data.Location),
		trimNullableString(data.URL),
		trimNullableString(data.SecondaryURL),
		trimNullableString(data.TertiaryURL),
		trimNullableString(data.QuaternaryURL),
		trimNullableString(data.QuinaryURL),
		trimNullableString(data.Title),
		trimNullableString(data.SecondaryTitle),
		trimNullableString(data.TertiaryTitle),
		trimNullableString(data.DistinguishedTitle),
		trimNullableString(data.ProfileSummary),
		trimNullableString(data.AcceptingStudents),
		trimNullableString(data.NotAcceptingStudents),
	} {
		if strings.TrimSpace(value) != "" {
			score++
		}
	}

	return score
}

func bestLocation(items []profileInformation) schema.Location {
	for _, item := range items {
		location := parseAPILocation(trimNullableString(item.Data.Location))
		if location.Building != "" || location.Room != "" {
			return location
		}
	}

	return schema.Location{}
}

func bestProfileURI(row profileIndexRow) string {
	if trimmed := strings.TrimSpace(row.URL); trimmed != "" {
		return trimmed
	}

	for _, info := range row.Information {
		for _, candidate := range []*string{info.Data.URL, info.Data.SecondaryURL, info.Data.TertiaryURL, info.Data.QuaternaryURL, info.Data.QuinaryURL} {
			trimmed := trimNullableString(candidate)
			if trimmed != "" {
				return trimmed
			}
		}
	}

	for _, candidate := range []string{row.APIURL} {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func bestImageURI(row profileIndexRow) string {
	if trimmed := strings.TrimSpace(row.ImageURL); trimmed != "" {
		return trimmed
	}

	for _, media := range row.Media {
		for _, key := range []string{"url", "image_url", "src", "uri"} {
			if raw, exists := media[key]; exists {
				if str, ok := raw.(string); ok {
					trimmed := strings.TrimSpace(str)
					if trimmed != "" {
						return trimmed
					}
				}
			}
		}
	}

	return ""
}

func mergeProfileProfessor(target, source *schema.Professor) {
	if target == nil || source == nil {
		return
	}
	if target.Titles == nil {
		target.Titles = []string{}
	}

	for _, title := range source.Titles {
		if !containsString(target.Titles, title) {
			target.Titles = append(target.Titles, title)
		}
	}

	if target.Email == "" {
		target.Email = source.Email
	}
	if target.Phone_number == "" {
		target.Phone_number = source.Phone_number
	}
	if target.Office.Building == "" && target.Office.Room == "" && target.Office.Map_uri == "" {
		target.Office = source.Office
	}
	if target.Profile_uri == "" {
		target.Profile_uri = source.Profile_uri
	}
	if target.Image_uri == "" {
		target.Image_uri = source.Image_uri
	}
	if target.Office_hours == nil {
		target.Office_hours = source.Office_hours
	}
	if target.Sections == nil {
		target.Sections = source.Sections
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func trimNullableString(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}