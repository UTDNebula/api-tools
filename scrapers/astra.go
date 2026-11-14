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
	astraMaxEventsPerDay = 5000
	astraBaseURL         = "https://www.aaiscloud.com/UTXDallas/~api/calendar/CalendarWeekGrid"
	astraFields          = "ActivityId,ActivityPk,ActivityName,ParentActivityId,ParentActivityName,MeetingType,Description,StartDate,EndDate,DayOfWeek,StartMinute,EndMinute,ActivityTypeCode,ResourceId,CampusName,BuildingCode,RoomNumber,RoomName,LocationName,InstitutionId,SectionId,SectionPk,IsExam,IsCrosslist,IsAllDay,IsPrivate,EventId,EventPk,CurrentState,NotAllowedUsageMask,UsageColor,UsageColorIsPrimary,EventTypeColor,MaxAttendance,ActualAttendance,Capacity"
	astraDelay           = 300 * time.Millisecond
)

type calendarResponse struct {
	Fields       string          `json:"fields"`
	Pos          int             `json:"pos"`
	Start        int             `json:"start"`
	TotalRecords int             `json:"totalRecords"`
	Data         json.RawMessage `json:"data"`
}

// ScrapeAstra iterates day-by-day through Astra events and persists the raw JSON output.
func ScrapeAstra(outDir string) error {
	// Ensure output folder
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	jar, _ := cookiejar.New(nil)

	client := &http.Client{
		Transport: http.DefaultTransport.(*http.Transport).Clone(),
		Timeout:   30 * time.Second,
		Jar:       jar,
	}

	data := make(map[string]*calendarResponse, 365)

	timeLocation := utils.GetUTDTimeZone()
	now := time.Now().In(timeLocation)
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, timeLocation)
	endDate := date.AddDate(1, 0, 0)

	for ; date.Before(endDate); date = date.AddDate(0, 0, 1) {
		dateStr := date.Format("2006-01-02")

		resp, err := queryAstra(client, dateStr)
		if err != nil {
			return fmt.Errorf("failed to query Astra for %s: %w", dateStr, err)
		}

		if resp.TotalRecords >= astraMaxEventsPerDay {
			return fmt.Errorf("WARNING: Max events per day exceeded for %s", dateStr)
		}

		data[dateStr] = resp
		time.Sleep(astraDelay)
	}

	log.Printf("Scraped Astra through %s!", endDate.AddDate(0, 0, -1).Format("2006-01-02"))

	if err := utils.WriteJSON(filepath.Join(outDir, "astraScraped.json"), data); err != nil {
		return fmt.Errorf("failed to write Astra scraped data to file : %w", err)
	}

	return nil
}

// queryAstra makes a request to the Astra Calendar API
func queryAstra(client *http.Client, dateStr string) (*calendarResponse, error) {
	log.Printf("Scraping %s...", dateStr)

	req, err := http.NewRequest("GET", astraBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	query := req.URL.Query()
	query.Set("_dc", strconv.FormatInt(time.Now().UnixMilli(), 10))
	query.Set("action", "GET")
	query.Set("start", "0")
	query.Set("limit", strconv.Itoa(astraMaxEventsPerDay))
	query.Set("isForWeekView", "false")
	query.Set("fields", astraFields)
	query.Set("filter", fmt.Sprintf(`(StartDate<="%sT23:59:59")&&(EndDate>="%sT00:00:00")`, dateStr, dateStr))
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
	defer func() {
		// draining the body so that the connection is released back to the pool
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("astra query failed with status code %s (Likely Rate Limited)", res.Status)
		}

		return nil, fmt.Errorf("astra query failed with status code %s", res.Status)
	}

	var calResponse calendarResponse
	if err := json.NewDecoder(res.Body).Decode(&calResponse); err != nil {
		return nil, fmt.Errorf("failed to decode Astra response: %w", err)
	}
	return &calResponse, nil
}
