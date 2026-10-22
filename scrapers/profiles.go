/*
	This file contains the code for the professor profile scraper.
*/

package scrapers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const profileBaseURL string = "https://profiles.utdallas.edu/api/v1"

const (
	profilesRawFileName      = "profiles_raw.json"
	profilesIndexRawFileName = "profiles_index_raw.json"
	profileSchoolsEnvVar     = "PROFILE_SCHOOLS"
)

const (
	profileBatchSize      = 25
	profileRequestTimeout = 30 * time.Second
	profileRequestDelay   = 200 * time.Millisecond
)

type profileIndexResponse struct {
	Count   int                  `json:"count"`
	Profile []profileIndexRecord `json:"profile"`
}

type profileIndexRecord struct {
	ID        int              `json:"id"`
	FullName  string           `json:"full_name"`
	FirstName string           `json:"first_name"`
	LastName  string           `json:"last_name"`
	Slug      string           `json:"slug"`
	Public    bool             `json:"public"`
	URL       string           `json:"url"`
	Name      string           `json:"name"`
	ImageURL  string           `json:"image_url"`
	APIURL    string           `json:"api_url"`
	Media     []map[string]any `json:"media"`
}

type profileRawRecord struct {
	ID        int                `json:"id"`
	FullName  string             `json:"full_name"`
	FirstName string             `json:"first_name"`
	LastName  string             `json:"last_name"`
	Slug      string             `json:"slug"`
	Public    bool               `json:"public"`
	URL       string             `json:"url"`
	Name      string             `json:"name"`
	ImageURL  string             `json:"image_url"`
	APIURL    string             `json:"api_url"`
	Media     []map[string]any   `json:"media"`
	Info      []profileInfoBlock `json:"information,omitempty"`
	Areas     []profileAreaBlock `json:"areas,omitempty"`
}

type profileInfoBlock struct {
	Data map[string]any `json:"data"`
}

type profileAreaBlock struct {
	Data map[string]any `json:"data"`
}

type profileDetailsResponse struct {
	Count   int                `json:"count"`
	Profile []profileRawRecord `json:"profile"`
}

type profileScrapeOutput struct {
	Count   int                `json:"count"`
	Profile []profileRawRecord `json:"profile"`
}

// ScrapeProfiles fetches the raw profile API response and writes it to disk.
func ScrapeProfiles(outDir string) {
	log.Print("Beginning profile scrape.")

	client := &http.Client{Timeout: profileRequestTimeout}

	indexResponse, err := fetchProfileIndex(client)
	if err != nil {
		log.Printf("Failed to retrieve profile index: %v", err)
		return
	}

	if len(indexResponse.Profile) == 0 {
		log.Print("Profile API returned no profiles.")
		return
	}

	schools := parseProfileSchoolsFromEnv()

	slugs := make([]string, 0, len(indexResponse.Profile))
	for _, row := range indexResponse.Profile {
		slug := strings.TrimSpace(row.Slug)
		if slug == "" {
			continue
		}
		slugs = append(slugs, slug)
	}
	slugs = dedupeStrings(slugs)

	if len(slugs) == 0 {
		log.Print("Profile API index contained no valid slugs.")
		return
	}

	log.Printf("Retrieved %d profile slugs.", len(slugs))

	detailedProfiles := make([]profileRawRecord, 0, len(slugs))
	if len(schools) > 0 {
		log.Printf("PROFILE_SCHOOLS configured with %d school codes. Pulling profile details by school.", len(schools))
		for i, school := range schools {
			schoolProfiles, fetchErr := fetchProfileDetailsForSchool(client, school)
			if fetchErr != nil {
				log.Printf("Failed to retrieve profile detail for school %s: %v", school, fetchErr)
				continue
			}

			detailedProfiles = append(detailedProfiles, schoolProfiles...)
			log.Printf("Fetched %d profile records for school %s.", len(schoolProfiles), school)

			if i < len(schools)-1 {
				time.Sleep(profileRequestDelay)
			}
		}
	} else {
		log.Printf("Pulling profile details by person slug in batches of %d.", profileBatchSize)
		for i := 0; i < len(slugs); i += profileBatchSize {
			end := i + profileBatchSize
			if end > len(slugs) {
				end = len(slugs)
			}

			batch := slugs[i:end]
			batchProfiles, fetchErr := fetchProfileDetails(client, batch)
			if fetchErr != nil {
				log.Printf("Failed to retrieve profile detail batch %d-%d: %v", i+1, end, fetchErr)
				continue
			}

			detailedProfiles = append(detailedProfiles, batchProfiles...)
			log.Printf("Fetched profile detail batch %d-%d (%d records).", i+1, end, len(batchProfiles))

			if end < len(slugs) {
				time.Sleep(profileRequestDelay)
			}
		}
	}

	detailedProfiles = dedupeProfiles(detailedProfiles)

	output := profileScrapeOutput{
		Count:   len(detailedProfiles),
		Profile: detailedProfiles,
	}

	if err := os.MkdirAll(outDir, 0777); err != nil {
		log.Printf("Failed to create output directory: %v", err)
		return
	}

	indexOutPath := filepath.Join(outDir, profilesIndexRawFileName)
	if err := writePrettyJSON(indexOutPath, indexResponse); err != nil {
		log.Printf("Failed to write profile index output file: %v", err)
		return
	}

	outPath := filepath.Join(outDir, profilesRawFileName)
	if err := writePrettyJSON(outPath, output); err != nil {
		log.Printf("Failed to write profile detail output file: %v", err)
		return
	}

	log.Printf("Wrote profile index to %s", indexOutPath)
	log.Printf("Wrote %d raw profiles to %s", output.Count, outPath)
}

func fetchProfileIndex(client *http.Client) (*profileIndexResponse, error) {
	req, err := http.NewRequest(http.MethodGet, profileBaseURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var decoded profileIndexResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	return &decoded, nil
}

func writePrettyJSON(path string, data any) error {
	fptr, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fptr.Close()

	encoder := json.NewEncoder(fptr)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return err
	}

	return nil
}

func fetchProfileDetails(client *http.Client, slugs []string) ([]profileRawRecord, error) {
	if len(slugs) == 0 {
		return []profileRawRecord{}, nil
	}

	requestURL := buildProfileDetailsRequestURL(slugs)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var decoded profileDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	return decoded.Profile, nil
}

func fetchProfileDetailsForSchool(client *http.Client, school string) ([]profileRawRecord, error) {
	requestURL := buildProfileSchoolRequestURL(school)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var decoded profileDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	return decoded.Profile, nil
}

func parseProfileSchoolsFromEnv() []string {
	return parseDelimitedValues(os.Getenv(profileSchoolsEnvVar))
}

func parseDelimitedValues(values string) []string {
	values = strings.TrimSpace(values)
	if values == "" {
		return []string{}
	}

	fields := strings.FieldsFunc(values, func(r rune) bool {
		switch r {
		case ';', ',', ' ', '\n', '\r', '\t':
			return true
		default:
			return false
		}
	})

	result := make([]string, 0, len(fields))
	for _, field := range fields {
		trimmed := strings.ToUpper(strings.TrimSpace(field))
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}

	return dedupeStrings(result)
}

func dedupeStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func buildProfileDetailsRequestURL(slugs []string) string {
	params := url.Values{}
	params.Set("person", strings.Join(slugs, ";"))
	params.Set("with_data", "1")
	params.Set("data_type", "information;areas")
	return fmt.Sprintf("%s?%s", profileBaseURL, params.Encode())
}

func buildProfileSchoolRequestURL(school string) string {
	params := url.Values{}
	params.Set("from_school", strings.TrimSpace(strings.ToUpper(school)))
	params.Set("with_data", "1")
	params.Set("data_type", "information;areas")
	return fmt.Sprintf("%s?%s", profileBaseURL, params.Encode())
}

func dedupeProfiles(values []profileRawRecord) []profileRawRecord {
	if len(values) < 2 {
		return values
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]profileRawRecord, 0, len(values))
	for _, value := range values {
		key := strings.TrimSpace(strings.ToLower(value.Slug))
		if key == "" {
			key = fmt.Sprintf("id:%d", value.ID)
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}

	return result
}
