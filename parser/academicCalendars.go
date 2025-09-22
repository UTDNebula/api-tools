package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/ledongthuc/pdf"
	"google.golang.org/genai"
)

// Store client to only create once
var once sync.Once
var geminiClient *genai.Client

var prompt = `Parse this PDF content and generate the following JSON schema.

{
  _id: %s,
  timeline: %s,
  sessions: [
    {
      name: string,
      begin: date string,
      last_registration: date string,
      late_registration: [date string, date string],
      census_day: date string,
      drop_deadlines {
        without_w: date string,
        undergrad_approval_required: date string, // use end date
        graduate_withdrawl_ends: date string,
      }
      end: date string,
      reading_days: [date string],
      exams: [date string, date string],
      final_grading_period: [date string, date string],
    }
  ],
  enrollment_opens: date string,
  schedule_planner_available: date string,
  online_add_swap_ends: date string,
  last_readmission: date string,
  last_from_waitlist: date string,
  midterms_due: date string,
  university_closings: [[date string, date string]], // for single days off use the same date string twice
  no_classes: [[date string, date string]],
}

- There will be 3 sessions for Fall and Spring and 4 sessions for Summer.
- You can determine the year for the dates based on the title. Be careful with Spring and Summer academic calendars as for example the 2025 one may have some earlier dates, such as registration, in 2024.
- Only use dates that are explicitly written in the PDF text. 
- Do not infer, estimate, or guess any date. 
- If a date is missing or unclear, return null for that field.

PDF Content:

%s`

func ParseAcademicCalendars(inDir string, outDir string) {
	pdf.DebugOn = true

	// Get sub folder from output folder
	outSubDir := filepath.Join(outDir, "academicCalendars")

	result := []schema.AcademicCalendar{}

	err := filepath.WalkDir(outSubDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() { // Is a file
			// Fall 2025 to 25F
			filename := filepath.Base(path)
			filename = filename[0 : len(filename)-4]
			filenameParts := strings.Split(filename, "-")
			name := filenameParts[1][len(filenameParts[1])-2 : len(filenameParts[1])]
			if strings.Contains(filenameParts[1], "Fall") {
				name = name + "F"
			} else if strings.Contains(filenameParts[1], "Spring") {
				name = name + "S"
			} else {
				name = name + "U"
			}
			timeline := filenameParts[0]

			// Parse
			academicCalendar, err := parsePdf(name, timeline, path)
			if err != nil {
				return err
			}
			result = append(result, academicCalendar)

			log.Printf("Parsed academic calendar %s!", filename)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	utils.WriteJSON(fmt.Sprintf("%s/academicCalendars.json", outDir), result)
}

func parsePdf(id string, timeline string, path string) (schema.AcademicCalendar, error) {
	content, err := readPdf(path)
	if err != nil {
		return schema.AcademicCalendar{}, err
	}

	geminiClient := getGeminiClient()

	// Build prompt
	promptFilled := fmt.Sprintf(prompt, id, timeline, content)

	// Send with default config
	response, err := geminiClient.Models.GenerateContent(context.Background(),
		"gemini-2.5-pro",
		genai.Text(promptFilled),
		&genai.GenerateContentConfig{},
	)
	if err != nil {
		return schema.AcademicCalendar{}, err
	}

	// Get response, remove backtick formatting
	jsonStr := strings.ReplaceAll(strings.ReplaceAll(response.Candidates[0].Content.Parts[0].Text, "```json", ""), "```", "")

	// Build struct
	var academicCalendar schema.AcademicCalendar
	err = json.Unmarshal([]byte(jsonStr), &academicCalendar)
	if err != nil {
		return schema.AcademicCalendar{}, err
	}

	return academicCalendar, nil
}

func readPdf(path string) (string, error) {
	// Open the PDF
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Make sure at least one page exists
	if r.NumPage() < 1 {
		return "", fmt.Errorf("no pages in PDF")
	}

	// Get the first page
	page := r.Page(1) // pages are 1-indexed
	if page.V.IsNull() {
		return "", fmt.Errorf("failed to read page 1")
	}

	// Read text
	var buf bytes.Buffer
	text := page.Content().Text
	for _, t := range text {
		buf.WriteString(t.S) // S is the actual string
	}

	return buf.String(), nil
}

// Create client only once
// Auth is from GOOGLE_GENAI_USE_VERTEXAI, GOOGLE_CLOUD_PROJECT and GOOGLE_APPLICATION_CREDENTIALS environment variables and service account JSON
func getGeminiClient() *genai.Client {
	once.Do(func() {
		// Create client
		var err error
		geminiClient, err = genai.NewClient(context.Background(),
			&genai.ClientConfig{
				Project:  "api-tools-451421",
				Location: "us-central1",
				Backend:  genai.BackendVertexAI,
			})
		if err != nil {
			panic(err)
		}
	})
	return geminiClient
}
