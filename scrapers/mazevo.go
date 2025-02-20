/*
	This file contains the code for the Mazevo scraper.
*/

package scrapers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/UTDNebula/api-tools/utils"
)

func ScrapeMazevo(outDir string) {
	apikey, err := utils.GetEnv("MAZEVO_API_KEY")
	if err != nil {
		panic(err)
	}

	// Make output folder
	err = os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	// Init http client
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	cli := &http.Client{Transport: tr}

	// Start on previous date to make sure we have today's data, regardless of what timezone the scraper is in
	date := time.Now()
	startDate := date.Add(time.Hour * -24).Format(time.RFC3339)
	endDate := date.Add(time.Hour * 24 * 365).Format(time.RFC3339)

	// Request events
	stringBody := ""
	{
		url := "https://east.mymazevo.com/api/PublicCalendar/GetCalendarEvents"
		requestBodyMap := map[string]string{
			"apiKey": apikey,
			"end":    endDate,
			"start":  startDate,
		}
		requestBodyBytes, _ := json.Marshal(requestBodyMap)
		requestBody := bytes.NewBuffer(requestBodyBytes)
		req, err := http.NewRequest("POST", url, requestBody)
		if err != nil {
			panic(err)
		}
		req.Header = http.Header{
			"Content-type": {"application/json"},
			"Accept":       {"application/json"},
		}
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
		stringBody = string(body)
	}

	// Write event data to output file
	fptr, err := os.Create(fmt.Sprintf("%s/mazevoReservations.json", outDir))
	if err != nil {
		panic(err)
	}
	_, err = fptr.Write([]byte(stringBody))
	if err != nil {
		panic(err)
	}
	fptr.Close()
}
