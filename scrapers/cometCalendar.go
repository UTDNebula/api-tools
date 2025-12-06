/*
	This file contains the code for the comet calendar events scraper.
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

const CAL_URL string = "https://calendar.utdallas.edu/api/2/events"

// RawEvent mirrors the nested event payload returned by the calendar API.
type RawEvent struct {
	Event map[string]any `json:"event"`
}

// APICalendarResponse models the calendar API pagination envelope.
type APICalendarResponse struct {
	Events []RawEvent        `json:"events"`
	Page   map[string]int    `json:"page"`
	Date   map[string]string `json:"date"`
}

// ScrapeCometCalendar retrieves calendar events through the API
func ScrapeCometCalendar(outDir string) {
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}
	client := http.Client{Timeout: 15 * time.Second}
	var calendarData APICalendarResponse

	// Get the total number of pages
	log.Printf("Getting the number of pages...")

	if err := callAndUnmarshal(&client, 0, &calendarData); err != nil {
		panic(err)
	}
	numPages := calendarData.Page["total"]
	log.Printf("The number of pages is %d!\n\n", numPages)

	var calendarEvents []schema.Event
	for page := range numPages {
		log.Printf("Scraping events of page %d...", page+1)
		if err := callAndUnmarshal(&client, page+1, &calendarData); err != nil {
			panic(err)
		}
		for _, rawEvent := range calendarData.Events {
			// Parse all necessary info
			startTime, endTime := getTime(rawEvent)
			eventTypes, targetAudiences, eventTopics := getFilters(rawEvent)
			departments, tags := getDepartmentsAndTags(rawEvent)
			contactInfo := getContactInfo(rawEvent)

			calendarEvents = append(calendarEvents, schema.Event{
				Id:                 primitive.NewObjectID(),
				Summary:            convert[string](rawEvent.Event["title"]),
				Location:           getEventLocation(rawEvent),
				StartTime:          startTime,
				EndTime:            endTime,
				Description:        convert[string](rawEvent.Event["description_text"]),
				EventType:          eventTypes,
				TargetAudience:     targetAudiences,
				Topic:              eventTopics,
				EventTags:          tags,
				EventWebsite:       convert[string](rawEvent.Event["url"]),
				Department:         departments,
				ContactName:        contactInfo[0],
				ContactEmail:       contactInfo[1],
				ContactPhoneNumber: contactInfo[2],
			})
		}

		log.Printf("Scraped events of page %d successfully!\n", page+1)
	}

	writePath := fmt.Sprintf("%s/cometCalendarScraped.json", outDir)
	if err := utils.WriteJSON(writePath, calendarEvents); err != nil {
		panic(err)
	}

	log.Printf("Finished scraping %d events successfully!\n\n", len(calendarEvents))
}

// scrapeAndUnmarshal fetches a calendar page and decodes it into data.
func callAndUnmarshal(client *http.Client, page int, data *APICalendarResponse) error {
	// Call API to get the byte data
	calendarUrl := fmt.Sprintf("%s?days=365&pp=100&page=%d", CAL_URL, page)
	request, err := http.NewRequest("GET", calendarUrl, nil)
	if err != nil {
		return err
	}
	request.Header = http.Header{
		"Content-type": {"application/json"},
		"Accept":       {"application/json"},
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response != nil && response.StatusCode != 200 {
		return fmt.Errorf("ERROR: Non-200 status is returned, %s", response.Status)
	}
	defer response.Body.Close()

	// Unmarshal bytes to the response data
	buffer := bytes.Buffer{}
	if _, err = buffer.ReadFrom(response.Body); err != nil {
		return err
	}
	if err = json.Unmarshal(buffer.Bytes(), &data); err != nil {
		return err
	}

	return nil
}

// getTime parses the start and end time of the event
func getTime(event RawEvent) (time.Time, time.Time) {
	instance := convert[map[string]any](
		convert[map[string]any](convert[[]any](event.Event["event_instances"])[0])["event_instance"])

	// Converts RFC3339 timestamp string to time.Time
	startTime, err := time.Parse(time.RFC3339, convert[string](instance["start"]))
	if err != nil {
		panic(err)
	}

	var endTime time.Time
	if convert[string](instance["end"]) != "" {
		endTime, err = time.Parse(time.RFC3339, convert[string](instance["end"]))
		if err != nil {
			panic(err)
		}
	} else {
		endTime = startTime
	}

	return startTime, endTime
}

// getEventLocation parses the location of the event
func getEventLocation(event RawEvent) string {
	building := convert[string](event.Event["location_name"])
	room := convert[string](event.Event["room_number"])
	location := strings.Trim(fmt.Sprintf("%s, %s", building, room), " ,")

	return location
}

// getFilters parses the types, topics, and target audiences
func getFilters(event RawEvent) ([]string, []string, []string) {
	eventTypes := []string{}
	targetAudiences := []string{}
	eventTopics := []string{}

	filters := convert[map[string]any](event.Event["filters"])

	rawTypes := convert[[]any](filters["event_types"])
	for _, rawType := range rawTypes {
		eventTypes = append(eventTypes, convert[string](convert[map[string]any](rawType)["name"]))
	}

	rawAudiences := convert[[]any](filters["event_target_audience"])
	for _, audience := range rawAudiences {
		targetAudiences = append(targetAudiences, convert[string](convert[map[string]any](audience)["name"]))
	}

	rawTopics := convert[[]any](filters["event_topic"])
	for _, topic := range rawTopics {
		eventTopics = append(eventTopics, convert[string](convert[map[string]any](topic)["name"]))
	}

	return eventTypes, targetAudiences, eventTopics
}

// getDepartmentsAndTags parses the departments, and tags
func getDepartmentsAndTags(event RawEvent) ([]string, []string) {
	departments := []string{}
	tags := []string{}

	rawTags := convert[[]any](event.Event["tags"])
	for _, tag := range rawTags {
		tags = append(tags, convert[string](tag))
	}

	rawDeparments := convert[[]any](event.Event["departments"])
	for _, deparment := range rawDeparments {
		departments = append(departments, convert[string](convert[map[string]any](deparment)["name"]))
	}

	return departments, tags
}

// getContactInfo parses the contact info.
func getContactInfo(event RawEvent) [3]string {
	// Note that some events won't have contact phone number
	contactInfo := [3]string{}

	rawContactInfo := convert[map[string]any](event.Event["custom_fields"])
	for i, infoField := range []string{
		"contact_information_name",
		"contact_information_email",
		"contact_information_phone",
	} {
		contactInfo[i] = convert[string](rawContactInfo[infoField])
	}

	return contactInfo
}

// convert() attempts to convert data into types for this scraper
func convert[T []any | map[string]any | string](data any) T {
	if newTypedData, ok := data.(T); ok {
		return newTypedData
	}
	var zeroValue T
	return zeroValue
}
