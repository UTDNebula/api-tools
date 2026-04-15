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

const COMET_CALENDAR_URL string = "https://calendar.utdallas.edu/api/2/events"

type EventInstance struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type FilterMap struct {
	Name string `json:"name"`
	Id   int    `json:"id"`
}

type Filters struct {
	Event_target_audience []FilterMap `json:"event_target_audience"`
	Event_topic           []FilterMap `json:"event_topic"`
	Event_types           []FilterMap `json:"event_types"`
}

type CustomFields struct {
	Contact_information_name  string `json:"contact_information_name"`
	Contact_information_email string `json:"contact_information_email"`
	Contact_information_phone string `json:"contact_information_phone"`
}

// Event mirrors the nested event payload returned by the calendar API.
type Event struct {
	Title            string   `json:"title"`
	Url              string   `json:"url"`
	Room_number      string   `json:"room_number"`
	Location_name    string   `json:"location_name"`
	Tags             []string `json:"tags"`
	Description_text string   `json:"description_text"`
	Event_instances  []struct {
		Event_instance EventInstance `json:"event_instance"`
	}
	Filters       Filters      `json:"filters"`
	Custom_fields CustomFields `json:"custom_fields"`
	Departments   []FilterMap  `json:"departments"`
}

// APICalendarResponse models the calendar API pagination envelope.
type APICalendarResponse struct {
	Events []struct {
		Event Event `json:"event"`
	} `json:"events"`
	Page map[string]int    `json:"page"`
	Date map[string]string `json:"date"`
}

// ScrapeCometCalendar retrieves calendar events through the API
func ScrapeCometCalendar(outDir string) {
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}
	client := http.Client{
		Timeout: 15 * time.Second,
	}
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
		for _, event := range calendarData.Events {
			// Parse all necessary info
			startTime, endTime, err := getTime(event.Event)
			if err != nil {
				panic(err)
			}
			if startTime.After(endTime) {
				fmt.Printf("start: %s, end: %s\n", startTime, endTime)
				continue
			}

			eventTypes, targetAudiences, eventTopics := getFilters(event.Event)
			departments := getDepartments(event.Event)

			calendarEvents = append(calendarEvents, schema.Event{
				Id:                 primitive.NewObjectID(),
				Summary:            event.Event.Title,
				Location:           getEventLocation(event.Event),
				StartTime:          startTime,
				EndTime:            endTime,
				Description:        event.Event.Description_text,
				EventType:          eventTypes,
				TargetAudience:     targetAudiences,
				Topic:              eventTopics,
				EventTags:          event.Event.Tags,
				EventWebsite:       event.Event.Url,
				Department:         departments,
				ContactName:        event.Event.Custom_fields.Contact_information_name,
				ContactEmail:       event.Event.Custom_fields.Contact_information_email,
				ContactPhoneNumber: event.Event.Custom_fields.Contact_information_phone,
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

// callAndUnmarshal fetches a calendar page and decodes it into data.
func callAndUnmarshal(client *http.Client, page int, data *APICalendarResponse) error {
	// Call API to get the byte data
	calendarUrl := fmt.Sprintf("%s?days=365&pp=100&page=%d", COMET_CALENDAR_URL, page)
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
	if err = json.Unmarshal(buffer.Bytes(), data); err != nil {
		return err
	}

	return nil
}

// getTime parses the start and end time of the event
func getTime(event Event) (time.Time, time.Time, error) {
	eventInstance := event.Event_instances[0].Event_instance

	// Converts RFC3339 timestamp string to time.Time
	startTime, err := time.Parse(time.RFC3339, eventInstance.Start)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	endTime := startTime
	if eventInstance.End != "" {
		endTime, err = time.Parse(time.RFC3339, eventInstance.End)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	return startTime, endTime, nil
}

// getEventLocation parses the location of the event
func getEventLocation(event Event) string {
	return strings.Trim(fmt.Sprintf("%s, %s", event.Location_name, event.Room_number), " ,")
}

// getFilters parses the types, topics, and target audiences
func getFilters(event Event) ([]string, []string, []string) {
	types := []string{}
	audiences := []string{}
	topics := []string{}

	for _, rawType := range event.Filters.Event_types {
		types = append(types, rawType.Name)
	}

	for _, rawAudience := range event.Filters.Event_target_audience {
		audiences = append(audiences, rawAudience.Name)
	}

	for _, rawTopic := range event.Filters.Event_topic {
		topics = append(topics, rawTopic.Name)
	}

	return types, audiences, topics
}

// getDepartmentsAndTags parses the departments, and tags
func getDepartments(event Event) []string {
	departments := []string{}
	for _, deparment := range event.Departments {
		departments = append(departments, deparment.Name)
	}

	return departments
}
