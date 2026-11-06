/*
	This file contains the code for the Astra scraper.
*/

package scrapers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/UTDNebula/api-tools/utils"
)

const (
	maxEventsPerDay = 5000
	baseURL         = "https://www.aaiscloud.com/UTXDallas/~api/calendar/CalendarWeekGrid"
	fields          = "ActivityId,ActivityPk,ActivityName,ParentActivityId,ParentActivityName,MeetingType,Description,StartDate,EndDate,DayOfWeek,StartMinute,EndMinute,ActivityTypeCode,ResourceId,CampusName,BuildingCode,RoomNumber,RoomName,LocationName,InstitutionId,SectionId,SectionPk,IsExam,IsCrosslist,IsAllDay,IsPrivate,EventId,EventPk,CurrentState,NotAllowedUsageMask,UsageColor,UsageColorIsPrimary,EventTypeColor,MaxAttendance,ActualAttendance,Capacity"
	astraMaxRetries = 3
)

type calendarResponse struct {
	Fields       string  `json:"fields"`
	Pos          int     `json:"pos"`
	Start        int     `json:"start"`
	TotalRecords int     `json:"totalRecords"`
	Data         [][]any `json:"data"`
}

// ScrapeAstra iterates day-by-day through Astra events and persists the raw JSON output.
func ScrapeAstra(outDir string) error {
	// Ensure output folder
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: 30 * time.Second,
		},
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	data := make(map[string]calendarResponse, 365)

	timeLocation, _ := time.LoadLocation("America/Chicago")
	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, timeLocation)

	for i := 0; i < 365; i++ {
		date = date.AddDate(0, 0, 1)

		var responseBody []byte
		err := utils.Retry(func() error {
			var err error
			responseBody, err = queryAstra(client, date)
			return err
		}, astraMaxRetries, astraRetryBackoff)

		if err != nil {
			return fmt.Errorf("failed to query Astra: %w", err)
		}

		calResponse := calendarResponse{}
		if err := json.Unmarshal(responseBody, &calResponse); err != nil {
			return fmt.Errorf("failed to unmarshal Astra response: %w", err)
		}

		if calResponse.TotalRecords >= maxEventsPerDay {
			return fmt.Errorf("ERROR: Max events per day exceeded")
		}

		data[date.Format("2006-01-02")] = calResponse
		time.Sleep(300 * time.Millisecond)
	}

	log.Printf("Scraped Astra up to %s!", date.Format("2006-01-02"))

	if err := utils.WriteJSON(filepath.Join(outDir, "astraScraped.json"), data); err != nil {
		return fmt.Errorf("failed to write Astra scraped data to file : %w", err)
	}

	return nil
}

func astraRetryBackoff(retry int) {
	delay := time.Duration(1<<(retry+1)) * time.Second
	log.Printf("Astra request failed, retrying in %s", delay)
	time.Sleep(delay)
}

// queryAstra makes a request to the Astra Calendar API and returns the raw JSON response.
func queryAstra(client *http.Client, day time.Time) ([]byte, error) {
	dateStr := day.Format("2006-01-02")

	log.Printf("Scraping %s...", dateStr)

	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to create request: %w", err)
	}

	query := req.URL.Query()
	query.Set("_dc", strconv.FormatInt(time.Now().UnixMilli(), 10))
	query.Set("action", "GET")
	query.Set("start", "0")
	query.Set("limit", strconv.Itoa(maxEventsPerDay))
	query.Set("isForWeekView", "false")
	query.Set("fields", fields)
	query.Set("filter", fmt.Sprintf(`(StartDate<="%sT23:00:00")&&(EndDate>="%sT00:00:00")`, dateStr, dateStr))
	query.Set("page", "1")
	query.Set("sortOrder", "+StartDate,+StartMinute")
	req.URL.RawQuery = query.Encode()

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:159.0) Gecko/20100101 Firefox/159.0")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://www.aaiscloud.com/UTXDallas/Default.aspx")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("astra query failed with status code %s (Likely Rate Limited)", res.Status)
		}

		return nil, fmt.Errorf("astra query failed with status code %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}
