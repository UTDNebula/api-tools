/*
Code requires having pdftotext installed: https://www.xpdfreader.com/pdftotext-man.html
apt-get install -y poppler-utils
I found all the Go programs for PDF text extraction were all either paid, had a
complicated installation process, or errored on one of the PDFs.
*/

package parser

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"google.golang.org/genai"
)

// What gets sent to Gemini, with the PDF content added
var academicCalendarPrompt = `Parse this PDF content and generate the following JSON schema.

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

- Use the ISO 8601 format for date strings (2006-01-02)
- There will be 3 sessions for Fall and Spring and 4 sessions for Summer.
- You can determine the year for the dates based on the title. Be careful with Spring and Summer academic calendars as for example the 2025 one may have some earlier dates, such as registration, in 2024.
- Only use dates that are explicitly written in the PDF text. 
- Do not infer, estimate, or guess any date. 
- If a date is missing or unclear, return null for that field.

PDF Content:

%s`

func ParseAcademicCalendars(inDir string, outDir string) {
	// Get sub folder from output folder
	inSubDir := filepath.Join(inDir, "academicCalendars")

	result := []schema.AcademicCalendar{}

	// Parallel requests
	numWorkers := 10
	jobs := make(chan string)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Start worker goroutines
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				log.Printf("Parsing %s...", filepath.Base(path))

				academicCalendar, err := parseAcademicCalendarPdf(path)
				if err != nil {
					if strings.Contains(err.Error(), "429") {
						// Exponential-ish backoff up to 60s for 429 rate limiting
						backoffs := []time.Duration{20 * time.Second, 40 * time.Second, 60 * time.Second}
						for _, delay := range backoffs {
							time.Sleep(delay)
							academicCalendar, err = parseAcademicCalendarPdf(path)
							if err == nil || !strings.Contains(err.Error(), "429") {
								break
							}
						}
					}

					if err != nil {
						panic(err)
					}
				}

				mu.Lock()
				result = append(result, academicCalendar)
				mu.Unlock()

				log.Printf("Parsed %s!", filepath.Base(path))
			}
		}()
	}

	err := filepath.WalkDir(inSubDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() { // Is a file
			jobs <- path
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	close(jobs)

	// Wait for workers to finish
	wg.Wait()

	utils.WriteJSON(fmt.Sprintf("%s/academicCalendars.json", outDir), result)
}

// Read a PDF, build a prompt for Gemini to parse it, check if it has already been asked in the cache, and ask Gemini if not
func parseAcademicCalendarPdf(path string) (schema.AcademicCalendar, error) {
	apiBucket, err := getAcademicCalendarBucket()
	if err != nil {
		return schema.AcademicCalendar{}, err
	}

	// "Fall 2025" to "25F"
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

	// Read PDF
	content, err := utils.ReadPdf(path, 1)
	if err != nil {
		return schema.AcademicCalendar{}, err
	}

	// Build prompt
	promptFilled := fmt.Sprintf(academicCalendarPrompt, name, timeline, content)

	// Check cache
	hashByte := sha256.Sum256([]byte(promptFilled))
	hash := hex.EncodeToString(hashByte[:]) + ".json"
	result, err := utils.CheckCache(hash, apiBucket)
	if err != nil {
		return schema.AcademicCalendar{}, err
	}

	// Skip AI if cache found
	if result != "" {
		log.Printf("Cache found for %s!", filename)
	} else {
		// Cache not found
		log.Printf("No cache for %s, asking Gemini.", filename)

		// AI
		geminiClient := utils.GetGeminiClient()

		// Response schema
		calendarSchema := utils.StructToSchema(reflect.TypeOf(schema.AcademicCalendar{}))

		// Send request with default config
		response, err := geminiClient.Models.GenerateContent(context.Background(),
			"gemini-2.5-pro",
			genai.Text(promptFilled),
			// Enforce response schema
			&genai.GenerateContentConfig{
				ResponseMIMEType: "application/json",
				ResponseSchema:   calendarSchema,
			},
		)
		if err != nil {
			return schema.AcademicCalendar{}, err
		}

		// Get response
		result = response.Candidates[0].Content.Parts[0].Text

		// Set cache for next time
		err = utils.SetCache(hash, result, apiBucket)
		if err != nil {
			return schema.AcademicCalendar{}, err
		}
	}

	// Build struct
	var academicCalendar schema.AcademicCalendar
	err = json.Unmarshal([]byte(result), &academicCalendar)
	if err != nil {
		return schema.AcademicCalendar{}, err
	}

	return academicCalendar, nil
}

// Get the storage bucket for the academic calendar cache
func getAcademicCalendarBucket() (string, error) {
	apiBucket, err := utils.GetEnv("NEBULA_API_STORAGE_BUCKET")
	if err != nil {
		return "", err
	}
	return apiBucket, nil
}
