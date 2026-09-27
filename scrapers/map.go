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

// API_KEY is the Concept3D API key observed from map.utdallas.edu traffic.
const API_KEY string = "0001085cc708b9cef47080f064612ca5"

// UTD_MAP_ID references the Concept3D map identifier for the UTD campus map.
const UTD_MAP_ID string = "1772"

// START_URL points to the Concept3D API host.
const START_URL string = "https://api.concept3d.com"

// END_URL appends the map and key query parameters for Concept3D requests.
const END_URL string = "/?map=" + UTD_MAP_ID + "&key=" + API_KEY

// ScrapeMapLocations downloads Concept3D responses and writes raw map data to disk.
func ScrapeMapLocations(outDir string) error {
	// Make output folder
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		return err
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
		return err
	}
	req.Header = http.Header{
		"Content-type": {"application/json"},
		"Accept":       {"application/json"},
	}
	res, err := cli.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		return fmt.Errorf("ERROR: Status was: %s", res.Status)
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return err
	}
	stringBody := string(body)

	log.Print("Scraped Map Locations!")

	// Write data to output file
	fptr, err := os.Create(fmt.Sprintf("%s/mapLocationsScraped.json", outDir))
	if err != nil {
		return err
	}
	defer fptr.Close()
	_, err = fptr.Write([]byte(stringBody))
	if err != nil {
		return err
	}
	return nil
}
