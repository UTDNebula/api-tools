/*
	This file contains the code for the events scraper.
*/

package scrapers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Structure of the API response
type RawEvent struct {
	Event map[string]interface{} `json:"event"`
}

type APICalendarResponse struct {
	Events []RawEvent        `json:"events"`
	Page   map[string]int    `json:"page"`
	Date   map[string]string `json:"date"`
}

// Get the calendar data through API instead of scraping from website
func ScrapeCalendar(outDir string) {
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}
	cli := http.Client{Timeout: 15 * time.Second}
	var calendarData APICalendarResponse

	// Get the total number of pages
	log.Printf("Getting the number of pages...")
	if err := scrapeAndUnmarshal(&cli, 0, &calendarData); err != nil {
		panic(err)
	}
	numPages := calendarData.Page["total"]
	log.Printf("The number of pages is %d!\n\n", numPages)

	var events []schema.Event
	for page := range numPages {
		log.Printf("Scraping events of page %d...", page+1)
		if err := scrapeAndUnmarshal(&cli, page+1, &calendarData); err != nil {
			panic(err)
		}
		log.Printf("Scraped events of page %d successfully!\n", page+1)

		log.Printf("Parsing the events of page %d...", page+1)
		for _, rawEvent := range calendarData.Events {
			// Parse the time
			eventInstance := toMap(toMap(toSlice(rawEvent.Event["event_instances"])[0])["event_instance"])
			startTime := parseTime(toString(eventInstance["start"]))
			endTime := startTime
			if toString(eventInstance["end"]) != "" {
				endTime = parseTime(toString(eventInstance["end"]))
			}

			// Parse location
			location := strings.Trim(fmt.Sprintf("%s, %s", toString(rawEvent.Event["location_name"]), toString(rawEvent.Event["room_number"])), " ,")

			// Parse the event types, event topic, and event target audience
			filters := toMap(rawEvent.Event["filters"])
			eventTypes := []string{}
			eventTopics := []string{}
			targetAudiences := []string{}

			rawTypes := toSlice(filters["event_types"])
			for _, rawType := range rawTypes {
				eventTypes = append(eventTypes, toString(toMap(rawType)["name"]))
			}

			rawAudiences := toSlice(filters["event_target_audience"])
			for _, audience := range rawAudiences {
				targetAudiences = append(targetAudiences, toString(toMap(audience)["name"]))
			}

			rawTopics := toSlice(filters["event_topic"])
			for _, topic := range rawTopics {
				eventTopics = append(eventTopics, toString(toMap(topic)["name"]))
			}

			// Parse the event departments, and tags
			departments := []string{}
			tags := []string{}

			rawTags := toSlice(rawEvent.Event["tags"])
			for _, tag := range rawTags {
				tags = append(tags, tag.(string))
			}

			rawDeparments := toSlice(rawEvent.Event["departments"])
			for _, deparment := range rawDeparments {
				departments = append(departments, toMap(deparment)["name"].(string))
			}

			// Parse the contact info, =ote that some events won't have contact phone number
			rawContactInfo := toMap(rawEvent.Event["custom_fields"])
			contactInfo := [3]string{}
			for i, infoField := range []string{
				"contact_information_name", "contact_information_email", "contact_information_phone",
			} {
				contactInfo[i] = toString(rawContactInfo[infoField])
			}

			events = append(events, schema.Event{
				Id:                 primitive.NewObjectID(),
				Summary:            toString(rawEvent.Event["title"]),
				Location:           location,
				StartTime:          startTime,
				EndTime:            endTime,
				Description:        toString(rawEvent.Event["description_text"]),
				EventType:          eventTypes,
				TargetAudience:     targetAudiences,
				Topic:              eventTopics,
				EventTags:          tags,
				EventWebsite:       toString(rawEvent.Event["url"]),
				Department:         departments,
				ContactName:        contactInfo[0],
				ContactEmail:       contactInfo[1],
				ContactPhoneNumber: contactInfo[2],
			})
		}
		log.Printf("Parsed the events of page %d successfully!\n\n", page+1)
	}

	if err := utils.WriteJSON(fmt.Sprintf("%s/events.json", outDir), events); err != nil {
		panic(err)
	}
	log.Printf("Finished parsing %d events successfully!\n\n", len(events))
}

// Scrape the data from the api and unmarshal it to response data
func scrapeAndUnmarshal(client *http.Client, page int, data *APICalendarResponse) error {
	// Call API to get the byte data
	calendarUrl := fmt.Sprintf("https://calendar.utdallas.edu/api/2/events?days=365&pp=100&page=%d", page)
	req, err := http.NewRequest("GET", calendarUrl, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	if res != nil && res.StatusCode != 200 {
		return fmt.Errorf("ERROR: Non-200 status is returned, %s", res.Status)
	}

	// Unmarshal bytes to the response data
	buffer := bytes.Buffer{}
	if _, err = buffer.ReadFrom(res.Body); err != nil {
		return err
	}
	res.Body.Close()
	if err = json.Unmarshal(buffer.Bytes(), &data); err != nil {
		return err
	}
	return nil
}

// Casting an interface{} to an slice of interface{}
func toSlice(data interface{}) []interface{} {
	if array, ok := data.([]interface{}); ok {
		return array
	}
	return nil
}

// Casting an interface{} to map from string to interface{}
func toMap(data interface{}) map[string]interface{} {
	if dataMap, ok := data.(map[string]interface{}); ok {
		return dataMap
	}
	return nil
}

// Casting an interface{} to string, if the data is nil then string is ""
func toString(data interface{}) string {
	if data != nil {
		if dataString, ok := data.(string); ok {
			return dataString
		}
	}
	return ""
}

// Parse string time
func parseTime(stringTime string) time.Time {
	parsedTime, err := time.Parse(time.RFC3339, stringTime)
	if err != nil {
		panic(err)
	}
	return parsedTime
}
