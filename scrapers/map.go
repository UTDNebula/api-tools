/*
	This file contains the code for the UTD map scraper.
*/

package scrapers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

//See API documentation https://devcms.concept3d.com/swagger/dist/ and https://api.concept3d.com/documentation/?map=1772&key=0001085cc708b9cef47080f064612ca5

// Found in dev tools on https://map.utdallas.edu/ in any call to https://api.concept3d.com/
const API_KEY string = "0001085cc708b9cef47080f064612ca5"

// Found in https://map.concept3d.com/?id=1772
const UTD_MAP_ID string = "1772"

const START_URL string = "https://api.concept3d.com"
const END_URL string = "/?map=" + UTD_MAP_ID + "&key=" + API_KEY

func ScrapeMapLocations(outDir string) {
	// Make output folder
	err := os.MkdirAll(outDir, 0777)
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

	// Request
	url := START_URL + "/locations" + END_URL
	req, err := http.NewRequest("GET", url, nil)
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
		log.Panicf("ERROR: Status was: %s", res.Status)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	res.Body.Close()
	stringBody := string(body)

	log.Print("Scraped Map Locations!")

	// Write data to output file
	fptr, err := os.Create(fmt.Sprintf("%s/mapLocationsScraped.json", outDir))
	if err != nil {
		panic(err)
	}
	_, err = fptr.Write([]byte(stringBody))
	if err != nil {
		panic(err)
	}
	fptr.Close()
}
