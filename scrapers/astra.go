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
)

func ScrapeAstra(outDir string) {

	// Load env vars
	if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
	}

	// Start chromedp
	chromedpCtx, cancel := utils.InitChromeDp()
	defer cancel()

	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	//days := []string

	// Init http client
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	cli := &http.Client{Transport: tr}

	astraHeaders := utils.RefreshAstraToken(chromedpCtx)
	time.Sleep(500 * time.Millisecond)

	//Request daily events
	date := time.Now().Format("2006-01-02")
	url := fmt.Sprintf("https://www.aaiscloud.com/UTXDallas/~api/calendar/CalendarWeekGrid?_dc=%d&action=GET&start=0&limit=5000&isForWeekView=false&fields=ActivityId,ActivityPk,ActivityName,ParentActivityId,ParentActivityName,MeetingType,Description,StartDate,EndDate,DayOfWeek,StartMinute,EndMinute,ActivityTypeCode,ResourceId,CampusName,BuildingCode,RoomNumber,RoomName,LocationName,InstitutionId,SectionId,SectionPk,IsExam,IsCrosslist,IsAllDay,IsPrivate,EventId,EventPk,CurrentState,NotAllowedUsageMask,UsageColor,UsageColorIsPrimary,EventTypeColor,MaxAttendance,ActualAttendance,Capacity&filter=(StartDate%%3C%%3D%%22%sT23%%3A00%%3A00%%22)%%26%%26(EndDate%%3E%%3D%%22%sT00%%3A00%%3A00%%22)&page=1", time.Now().UnixMilli(), date, date)
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

	// Write event data to output file
	fptr, err := os.Create(fmt.Sprintf("%s/reservations.json", outDir))
	if err != nil {
		panic(err)
	}
	_, err = fptr.Write(body)
	if err != nil {
		panic(err)
	}
	fptr.Close()
}
