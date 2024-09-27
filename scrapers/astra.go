/*
	This file contains the code for the Astra scraper.
*/

package scrapers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/joho/godotenv"
	"github.com/valyala/fastjson"
)

func ScrapeAstra(outDir string) {

	// Load env vars
	if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
	}

	// Start chromedp
	chromedpCtx, cancel := utils.InitChromeDp()
	defer cancel()

	// Make output folder
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	days := "{"       // String JSON for storing results by day
	firstLoop := true // To avoid adding a comma to the JSON on the first loop

	// Init http client
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	cli := &http.Client{Transport: tr}

	astraHeaders := utils.RefreshAstraToken(chromedpCtx)
	time.Sleep(500 * time.Millisecond)

	// Starting date
	date := time.Now()

	// Stop condition
	lt10EventsCount := 0

	// Run until 90 days of no events
	for lt10EventsCount < 90 {
		formattedDate := date.Format("2006-01-02")
		log.Printf("Scraping %s...", formattedDate)

		// Request daily events
		url := fmt.Sprintf("https://www.aaiscloud.com/UTXDallas/~api/calendar/CalendarWeekGrid?_dc=%d&action=GET&start=0&limit=5000&isForWeekView=false&fields=ActivityId,ActivityPk,ActivityName,ParentActivityId,ParentActivityName,MeetingType,Description,StartDate,EndDate,DayOfWeek,StartMinute,EndMinute,ActivityTypeCode,ResourceId,CampusName,BuildingCode,RoomNumber,RoomName,LocationName,InstitutionId,SectionId,SectionPk,IsExam,IsCrosslist,IsAllDay,IsPrivate,EventId,EventPk,CurrentState,NotAllowedUsageMask,UsageColor,UsageColorIsPrimary,EventTypeColor,MaxAttendance,ActualAttendance,Capacity&filter=(StartDate%%3C%%3D%%22%sT23%%3A00%%3A00%%22)%%26%%26(EndDate%%3E%%3D%%22%sT00%%3A00%%3A00%%22)&page=1&sortOrder=%%2BStartDate,%%2BStartMinute", time.Now().UnixMilli(), formattedDate, formattedDate)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			panic(err)
		}
		req.Header = astraHeaders
		res, err := cli.Do(req)
		if err != nil {
			panic(err)
		}
		if res.StatusCode != 200 {
			log.Panicf("ERROR: Status was: %s\nIf the status is 404, you've likely been IP ratelimited!", res.Status)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}
		stringBody := string(body)

		// Check for no events
		if fastjson.GetInt(body, "totalRecords") < 10 {
			lt10EventsCount += 1
			if lt10EventsCount > 30 {
				log.Printf("There have been %d days in a row with fewer than 10 events.", lt10EventsCount)
			}
		} else {
			lt10EventsCount = 0
		}

		// Add to record
		comma := ","
		if firstLoop {
			comma = ""
			firstLoop = false
		}
		days = fmt.Sprintf("%s%s\"%s\":%s", days, comma, formattedDate, stringBody)
		date = date.Add(time.Hour * 24)
	}

	// Write event data to output file
	days = fmt.Sprintf("%s}", days)
	fptr, err := os.Create(fmt.Sprintf("%s/reservations.json", outDir))
	if err != nil {
		panic(err)
	}
	_, err = fptr.Write([]byte(days))
	if err != nil {
		panic(err)
	}
	fptr.Close()
}
