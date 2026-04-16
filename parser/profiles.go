package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const profilesRawFileName = "profiles.json"

var (
	apiPrimaryLocationRegex  = regexp.MustCompile(`^(\w+)\s+(\d+\.\d{3}[A-Za-z]?)$`)
	apiFallbackLocationRegex = regexp.MustCompile(`^([A-Za-z]+)(\d+)\.?([\d]{3}[A-Za-z]?)$`)
)

type profileIndexResponse struct {
	Count   int               `json:"count"`
	Profile []profileIndexRow `json:"profile"`
}

type profileIndexRow struct {
	ID          int                  `json:"id"`
	FullName    string               `json:"full_name"`
	FirstName   string               `json:"first_name"`
	LastName    string               `json:"last_name"`
	Slug        string               `json:"slug"`
	Public      bool                 `json:"public"`
	URL         string               `json:"url"`
	Name        string               `json:"name"`
	ImageURL    string               `json:"image_url"`
	APIURL      string               `json:"api_url"`
	Media       []map[string]any     `json:"media"`
	Information []profileInformation `json:"information"`
	Areas       []profileArea        `json:"areas"`
}

type profileInformation struct {
	Data profileInformationData `json:"data"`
}

type profileInformationData struct {
	URL                  string `json:"url"`
	SecondaryURL         string `json:"secondary_url"`
	TertiaryURL          string `json:"tertiary_url"`
	QuaternaryURL        string `json:"quaternary_url"`
	QuinaryURL           string `json:"quinary_url"`
	Email                string `json:"email"`
	Phone                string `json:"phone"`
	Title                string `json:"title"`
	SecondaryTitle       string `json:"secondary_title"`
	TertiaryTitle        string `json:"tertiary_title"`
	DistinguishedTitle   string `json:"distinguished_title"`
	Location             string `json:"location"`
	ProfileSummary       string `json:"profile_summary"`
	AcceptingStudents    string `json:"accepting_students"`
	NotAcceptingStudents string `json:"not_accepting_students"`
}

type profileArea struct {
	Data profileAreaData `json:"data"`
}

type profileAreaData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// LoadProfiles reads scraped profile API data and populates the package maps.
func LoadProfiles(inDir string) bool {
	path := fmt.Sprintf("%s/%s", inDir, profilesRawFileName)
	fptr, err := os.Open(path)
	if err != nil {
		return false
	}
	defer fptr.Close()

	payload, err := io.ReadAll(fptr)
	if err != nil {
		log.Printf("Failed to read profiles JSON: %v", err)
		return false
	}

	rows, err := decodeProfileRows(payload)
	if err != nil {
		log.Printf("Failed to decode profiles JSON: %v", err)
		return false
	}

	loadedCount := 0
	for _, row := range rows {
		if !row.Public {
			continue
		}

		prof := buildProfessorFromRow(row)
		if prof == nil {
			continue
		}

		professorKey := prof.First_name + prof.Last_name
		if _, exists := Professors[professorKey]; exists {
			continue
		}
		Professors[professorKey] = prof
		ProfessorIDMap[prof.Id] = professorKey
		loadedCount++
	}

	log.Printf("Loaded %d profiles from %s.", loadedCount, profilesRawFileName)
	return true
}

func decodeProfileRows(payload []byte) ([]profileIndexRow, error) {
	var rows []profileIndexRow
	if err := json.Unmarshal(payload, &rows); err == nil {
		return rows, nil
	}

	var response profileIndexResponse
	if err := json.Unmarshal(payload, &response); err == nil {
		return response.Profile, nil
	}

	return nil, fmt.Errorf("unsupported profiles JSON shape")
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

	titles := collectTitles(row)
	info := bestInformationData(row.Information)

	prof := &schema.Professor{}
	prof.Id = primitive.NewObjectID()
	prof.First_name = firstName
	prof.Last_name = lastName
	prof.Titles = titles
	prof.Email = strings.TrimSpace(info.Email)
	prof.Phone_number = strings.TrimSpace(info.Phone)
	prof.Office = bestLocation(row.Information)
	prof.Profile_uri = bestProfileURI(row)
	prof.Image_uri = bestImageURI(row)
	prof.Office_hours = []schema.Meeting{}
	prof.Sections = []primitive.ObjectID{}

	return prof
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
		for _, candidate := range []string{info.Data.Title, info.Data.SecondaryTitle, info.Data.TertiaryTitle, info.Data.DistinguishedTitle} {
			trimmed := strings.TrimSpace(candidate)
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
		data.Email,
		data.Phone,
		data.Location,
		data.URL,
		data.SecondaryURL,
		data.TertiaryURL,
		data.QuaternaryURL,
		data.QuinaryURL,
		data.Title,
		data.SecondaryTitle,
		data.TertiaryTitle,
		data.DistinguishedTitle,
		data.ProfileSummary,
		data.AcceptingStudents,
		data.NotAcceptingStudents,
	} {
		if strings.TrimSpace(value) != "" {
			score++
		}
	}

	return score
}

func bestLocation(items []profileInformation) schema.Location {
	for _, item := range items {
		location := parseAPILocation(item.Data.Location)
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
		for _, candidate := range []string{info.Data.URL, info.Data.SecondaryURL, info.Data.TertiaryURL, info.Data.QuaternaryURL, info.Data.QuinaryURL} {
			trimmed := strings.TrimSpace(candidate)
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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
