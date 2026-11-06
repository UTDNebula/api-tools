/*
	This file contains the code for the Astra scraper.
*/

package scrapers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/valyala/fastjson"
)

// MAX_EVENTS_PER_DAY caps the Astra API results to guard against truncated responses.
var MAX_EVENTS_PER_DAY = 5000

// ScrapeAstra iterates day-by-day through Astra events and persists the raw JSON output.
func ScrapeAstra(outDir string) {
	// Make output folder
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	days := "{"       // String JSON for storing results by day
	firstLoop := true // To avoid adding a comma to the JSON on the first loop

	// Init http client
	jar, _ := cookiejar.New(nil)
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	cli := &http.Client{Transport: tr, Jar: jar}

	date := time.Now()
	endDate := date.Add(time.Hour * 24 * 365)

	// Start on previous date to make sure we have today's data, regardless of what timezone the scraper is in
	date = date.Add(time.Hour * -24)

	// Run for a year
	for ; date.Before(endDate); date = date.Add(time.Hour * 24) {
		formattedDate := date.Format("2006-01-02")
		log.Printf("Scraping %s...", formattedDate)

		// Request daily events
		url := fmt.Sprintf("https://www.aaiscloud.com/UTXDallas/~api/calendar/CalendarWeekGrid?_dc=%d&action=GET&start=0&limit=%d&isForWeekView=false&fields=ActivityId,ActivityPk,ActivityName,ParentActivityId,ParentActivityName,MeetingType,Description,StartDate,EndDate,DayOfWeek,StartMinute,EndMinute,ActivityTypeCode,ResourceId,CampusName,BuildingCode,RoomNumber,RoomName,LocationName,InstitutionId,SectionId,SectionPk,IsExam,IsCrosslist,IsAllDay,IsPrivate,EventId,EventPk,CurrentState,NotAllowedUsageMask,UsageColor,UsageColorIsPrimary,EventTypeColor,MaxAttendance,ActualAttendance,Capacity&filter=(StartDate%%3C%%3D%%22%sT23%%3A00%%3A00%%22)%%26%%26(EndDate%%3E%%3D%%22%sT00%%3A00%%3A00%%22)&page=1&sortOrder=%%2BStartDate,%%2BStartMinute", time.Now().UnixMilli(), MAX_EVENTS_PER_DAY, formattedDate, formattedDate)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			panic(err)
		}

		// Set the necessary headers directly to bypass the old chromedp login
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:159.0) Gecko/20100101 Firefox/159.0")
		req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("Referer", "https://www.aaiscloud.com/UTXDallas/Default.aspx")

		res, err := cli.Do(req)
		if err != nil {
			panic(err)
		}

		if res.StatusCode != 200 {
			log.Panicf("ERROR: Status was: %s\nIf the status is 404, you've likely been IP ratelimited!", res.Status)
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}
		res.Body.Close()
		stringBody := string(body)

		// Check for no events
		numEvents := fastjson.GetInt(body, "totalRecords")
		if numEvents >= MAX_EVENTS_PER_DAY {
			log.Panic("ERROR: Max events per day exceeded!")
		}

		// Add to record
		comma := ","
		if firstLoop {
			comma = ""
			firstLoop = false
		}
		days = fmt.Sprintf("%s%s\"%s\":%s", days, comma, formattedDate, stringBody)
	}

	log.Printf("Scraped Astra up to %s!", date.Format("2006-01-02"))

	// Close the JSON object string
	days = fmt.Sprintf("%s}", days)

	// Format the raw JSON string with tabs for indentation
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, []byte(days), "", "\t")
	if err != nil {
		panic(err)
	}

	// Write formatted event data to output file
	fptr, err := os.Create(fmt.Sprintf("%s/astraScraped.json", outDir))
	if err != nil {
		panic(err)
	}
	defer fptr.Close() // Best practice to use defer for file closing

	_, err = fptr.Write(prettyJSON.Bytes())
	if err != nil {
		panic(err)
	}
}
