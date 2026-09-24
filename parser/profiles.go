package parser

import (
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const profilesRawFileName = "profiles.json"

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
	URL                  *string `json:"url"`
	SecondaryURL         *string `json:"secondary_url"`
	TertiaryURL          *string `json:"tertiary_url"`
	QuaternaryURL        *string `json:"quaternary_url"`
	QuinaryURL           *string `json:"quinary_url"`
	Email                *string `json:"email"`
	Phone                *string `json:"phone"`
	Title                *string `json:"title"`
	SecondaryTitle       *string `json:"secondary_title"`
	TertiaryTitle        *string `json:"tertiary_title"`
	DistinguishedTitle   *string `json:"distinguished_title"`
	Location             *string `json:"location"`
	ProfileSummary       *string `json:"profile_summary"`
	AcceptingStudents    *string `json:"accepting_students"`
	NotAcceptingStudents *string `json:"not_accepting_students"`
}

type profileArea struct {
	Data profileAreaData `json:"data"`
}

type profileAreaData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
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

// syncProfessorSectionLinks ensures professor->section links match parsed section references.
// Course connections are derived transitively through each section's Course_reference.
func syncProfessorSectionLinks() {
	if len(Professors) == 0 || len(Sections) == 0 {
		return
	}

	for _, prof := range Professors {
		if prof == nil {
			continue
		}
		if prof.Sections == nil {
			prof.Sections = []primitive.ObjectID{}
		}
	}

	for sectionID, section := range Sections {
		if section == nil {
			continue
		}

		for _, profID := range section.Professors {
			profKey, ok := ProfessorIDMap[profID]
			if !ok {
				continue
			}
			prof, exists := Professors[profKey]
			if !exists || prof == nil {
				continue
			}
			if !containsObjectID(prof.Sections, sectionID) {
				prof.Sections = append(prof.Sections, sectionID)
			}
		}
	}

	for _, prof := range Professors {
		if prof == nil {
			continue
		}
		prof.Sections = dedupeObjectIDs(prof.Sections)
	}
}

func containsObjectID(values []primitive.ObjectID, target primitive.ObjectID) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

func dedupeObjectIDs(values []primitive.ObjectID) []primitive.ObjectID {
	if len(values) < 2 {
		return values
	}

	seen := make(map[primitive.ObjectID]struct{}, len(values))
	result := make([]primitive.ObjectID, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
