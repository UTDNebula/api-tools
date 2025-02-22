package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

type SourceData struct {
	Bookings []map[string]interface{} `json:"bookings"`
}

func ParseMazevo(inDir string, outDir string) {

	mazevoFile, err := os.ReadFile(inDir + "/mazevoScraped.json")
	if err != nil {
		panic(err)
	}

	var rawData SourceData
	err = json.Unmarshal(mazevoFile, &rawData)
	if err != nil {
		panic(err)
	}

	multiBuildingMap := make(map[string]map[string]map[string][]schema.MazevoEvent)

	for _, rawEvent := range rawData.Bookings {
		datePtr := pullString(rawEvent["dateTimeStart"])
		if datePtr == nil {
			continue
		}
		date := (*datePtr)[:10]
		building := pullString(rawEvent["buildingDescription"])
		room := pullString(rawEvent["roomDescription"])
		event := schema.MazevoEvent{
			EventName:         pullString(rawEvent["eventName"]),
			OrganizationName:  pullString(rawEvent["organizationName"]),
			ContactName:       pullString(rawEvent["contactName"]),
			SetupMinutes:      pullInt(rawEvent["setupMinutes"]),
			DateTimeStart:     pullString(rawEvent["dateTimeStart"]),
			DateTimeEnd:       pullString(rawEvent["dateTimeEnd"]),
			TeardownMinutes:   pullInt(rawEvent["teardownMinutes"]),
			StatusDescription: pullString(rawEvent["statusDescription"]),
			StatusColor:       pullString(rawEvent["statusColor"]),
		}

		if building == nil || room == nil || *(building) == "" || *(room) == "" {
			continue
		}
		*building = strings.TrimSpace(*building)

		if _, exists := multiBuildingMap[date]; !exists {
			multiBuildingMap[date] = make(map[string]map[string][]schema.MazevoEvent)
		}
		if _, exists := multiBuildingMap[date][*building]; !exists {
			multiBuildingMap[date][*building] = make(map[string][]schema.MazevoEvent)
		}
		multiBuildingMap[date][*building][*room] = append(multiBuildingMap[date][*building][*room], event)
	}

	var result []schema.MultiBuildingEvents[schema.MazevoEvent]

	for date, buildings := range multiBuildingMap {
		var buildingList []schema.SingleBuildingEvents[schema.MazevoEvent]
		for building, rooms := range buildings {
			var roomList []schema.RoomEvents[schema.MazevoEvent]
			for room, events := range rooms {
				roomList = append(roomList, schema.RoomEvents[schema.MazevoEvent]{
					Room:   room,
					Events: events,
				})
			}
			buildingList = append(buildingList, schema.SingleBuildingEvents[schema.MazevoEvent]{
				Building: building,
				Rooms:    roomList,
			})
		}
		result = append(result, schema.MultiBuildingEvents[schema.MazevoEvent]{
			Date:      date,
			Buildings: buildingList,
		})
	}

	log.Print("Parsed Mazevo!")

	utils.WriteJSON(fmt.Sprintf("%s/mazevo.json", outDir), result)
}

func pullString(value interface{}) *string {
	if parsed, ok := value.(string); ok {
		return &parsed
	}
	return nil
}

func pullInt(value interface{}) *float64 {
	if parsed, ok := value.(float64); ok {
		return &parsed
	}
	return nil
}
