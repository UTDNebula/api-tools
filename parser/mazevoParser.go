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

var buildingRenames = map[string]string{
	"Student Union":                   "SU",
	"Student Services Addition (SSA)": "SSA",
}

// SourceData represents the Mazevo API response containing booking records.
type SourceData struct {
	Bookings []map[string]interface{} `json:"bookings"`
}

// ParseMazevo reads Mazevo scrape output and emits normalized multi-building event JSON.
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
		datePtr := utils.ConvertFromInterface[string](rawEvent["dateTimeStart"])
		if datePtr == nil {
			continue
		}
		date := (*datePtr)[:10]
		building := utils.ConvertFromInterface[string](rawEvent["buildingDescription"])
		room := utils.ConvertFromInterface[string](rawEvent["roomDescription"])
		event := schema.MazevoEvent{
			EventName:         utils.ConvertFromInterface[string](rawEvent["eventName"]),
			OrganizationName:  utils.ConvertFromInterface[string](rawEvent["organizationName"]),
			ContactName:       utils.ConvertFromInterface[string](rawEvent["contactName"]),
			SetupMinutes:      utils.ConvertFromInterface[float64](rawEvent["setupMinutes"]),
			DateTimeStart:     utils.ConvertFromInterface[string](rawEvent["dateTimeStart"]),
			DateTimeEnd:       utils.ConvertFromInterface[string](rawEvent["dateTimeEnd"]),
			TeardownMinutes:   utils.ConvertFromInterface[float64](rawEvent["teardownMinutes"]),
			StatusDescription: utils.ConvertFromInterface[string](rawEvent["statusDescription"]),
			StatusColor:       utils.ConvertFromInterface[string](rawEvent["statusColor"]),
		}

		if building == nil || room == nil || *(building) == "" || *(room) == "" {
			continue
		}
		*building = strings.TrimSpace(*building)
		for key, value := range buildingRenames {
			if *building == key {
				*building = value
			}
			if strings.HasPrefix(*room, value+" ") {
				*room = strings.TrimPrefix(*room, value+" ")
			}
		}

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
