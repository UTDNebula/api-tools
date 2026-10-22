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
	profilesRawFileName = "profiles.json"
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
	Media     []profileMedia   `json:"media"`
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
	Media     []profileMedia     `json:"media"`
	Information []profileInfoBlock `json:"information"`
	Areas     []profileAreaBlock `json:"areas"`
}

type profileMedia struct {
	ID                  int              `json:"id"`
	ModelID             int              `json:"model_id"`
	UUID                string           `json:"uuid"`
	ModelType           string           `json:"model_type"`
	CollectionName      string           `json:"collection_name"`
	Name                string           `json:"name"`
	FileName            string           `json:"file_name"`
	MimeType            string           `json:"mime_type"`
	Disk                string           `json:"disk"`
	ConversionsDisk     string           `json:"conversions_disk"`
	Size                int              `json:"size"`
	Manipulations       []any            `json:"manipulations"`
	CustomProperties     []any            `json:"custom_properties"`
	GeneratedConversions profileGeneratedConversions `json:"generated_conversions"`
	ResponsiveImages    any              `json:"responsive_images"`
	OrderColumn         int              `json:"order_column"`
	CreatedAt           string           `json:"created_at"`
	UpdatedAt           string           `json:"updated_at"`
	OriginalURL         string           `json:"original_url"`
	PreviewURL          string           `json:"preview_url"`
}

type profileGeneratedConversions struct {
	Large bool `json:"large"`
	Thumb bool `json:"thumb"`
	Medium bool `json:"medium"`
}

type profileInformationData struct {
	URL                     *string `json:"url"`
	Email                   *string `json:"email"`
	Phone                   *string `json:"phone"`
	Title                   *string `json:"title"`
	ORCID                   *string `json:"orc_id"`
	Location                *string `json:"location"`
	URLName                 *string `json:"url_name"`
	QuinaryURL              *string `json:"quinary_url"`
	FancyHeader             *string `json:"fancy_header"`
	TertiaryURL             *string `json:"tertiary_url"`
	SecondaryURL            *string `json:"secondary_url"`
	ORCIDManaged            *string `json:"orc_id_managed"`
	QuaternaryURL           *string `json:"quaternary_url"`
	TertiaryTitle           *string `json:"tertiary_title"`
	ProfileSummary          *string `json:"profile_summary"`
	SecondaryTitle          *string `json:"secondary_title"`
	QuinaryURLName          *string `json:"quinary_url_name"`
	TertiaryURLName         *string `json:"tertiary_url_name"`
	AcceptingStudents       *string `json:"accepting_students"`
	FancyHeaderRight        *string `json:"fancy_header_right"`
	SecondaryURLName        *string `json:"secondary_url_name"`
	DistinguishedTitle      *string `json:"distinguished_title"`
	QuaternaryURLName       *string `json:"quaternary_url_name"`
	NotAcceptingStudents    *string `json:"not_accepting_students"`
	AcceptingGradStudents   *string `json:"accepting_grad_students"`
	ShowAcceptingStudents   *string `json:"show_accepting_students"`
	NotAcceptingGradStudents *string `json:"not_accepting_grad_students"`
	ShowNotAcceptingStudents *string `json:"show_not_accepting_students"`
}

type profileInfoBlock struct {
	ID        int                   `json:"id"`
	ProfileID int                   `json:"profile_id"`
	Type      string                `json:"type"`
	SortOrder int                   `json:"sort_order"`
	Data      profileInformationData `json:"data"`
	Public    bool                  `json:"public"`
	CreatedAt string                `json:"created_at"`
	UpdatedAt string                `json:"updated_at"`
}

type profileAreaData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type profileAreaBlock struct {
	ID        int             `json:"id"`
	ProfileID int             `json:"profile_id"`
	Type      string          `json:"type"`
	SortOrder int             `json:"sort_order"`
	Data      profileAreaData `json:"data"`
	Public    bool            `json:"public"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

type profileDetailsResponse struct {
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

	detailedProfiles = dedupeProfiles(detailedProfiles)
	for i := range detailedProfiles {
		if detailedProfiles[i].Media == nil {
			detailedProfiles[i].Media = []profileMedia{}
		}
		if detailedProfiles[i].Information == nil {
			detailedProfiles[i].Information = []profileInfoBlock{}
		}
		if detailedProfiles[i].Areas == nil {
			detailedProfiles[i].Areas = []profileAreaBlock{}
		}
		for j := range detailedProfiles[i].Media {
			if detailedProfiles[i].Media[j].Manipulations == nil {
				detailedProfiles[i].Media[j].Manipulations = []any{}
			}
			if detailedProfiles[i].Media[j].CustomProperties == nil {
				detailedProfiles[i].Media[j].CustomProperties = []any{}
			}
		}
	}

	if err := os.MkdirAll(outDir, 0777); err != nil {
		log.Printf("Failed to create output directory: %v", err)
		return
	}

	outPath := filepath.Join(outDir, profilesRawFileName)
	detailOutput := profileDetailsResponse{Count: len(detailedProfiles), Profile: detailedProfiles}
	if err := writePrettyJSON(outPath, detailOutput); err != nil {
		log.Printf("Failed to write profile detail output file: %v", err)
		return
	}

	log.Printf("Wrote %d raw profiles to %s", len(detailedProfiles), outPath)
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
