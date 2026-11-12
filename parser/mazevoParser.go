package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/rest/schema"
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
	inDir = filepath.Join(inDir, "mazevo")
	dir, err := os.ReadDir(inDir)
	if err != nil {
		return
	}

	multiBuildingMap := make(map[string]map[string]map[string][]schema.MazevoEvent)

	for _, file := range dir {
		// Skip subdirectories
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(inDir, file.Name())
		fileBytes, err := os.ReadFile(filePath)
		if err != nil {
			panic(err)
		}

		// Unmarshal decodes the scraped JSON data
		var rawData SourceData
		err = json.Unmarshal(fileBytes, &rawData)
		if err != nil {
			panic(err)
		}

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
				if after, ok := strings.CutPrefix(*room, value+" "); ok {
					*room = after
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
	}

	// Nested slices (date -> building -> room -> events) using schema.
	var result []schema.MultiBuildingEvents[schema.MazevoEvent]

	// NOTE: In Go, any iterating through a map is done in a random order.
	// This applies to the nested for loop below
	// If you require any sorting of this data, put the keys in a slice, sort the slice
	// and loop over that instead.
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

	// sort result (a nested array of slices) using slices.Sortfunc
	slices.SortFunc(result, func(a, b schema.MultiBuildingEvents[schema.MazevoEvent]) int {
		// sorts by date, ascending
		return strings.Compare(a.Date, b.Date)
	})

	for i := range result {
		slices.SortFunc(result[i].Buildings, func(a, b schema.SingleBuildingEvents[schema.MazevoEvent]) int {
			// sorts alphabetically
			return strings.Compare(a.Building, b.Building)
		})

		for j := range result[i].Buildings {
			slices.SortFunc(result[i].Buildings[j].Rooms, func(c, d schema.RoomEvents[schema.MazevoEvent]) int {
				// sorts alphabetically
				return strings.Compare(c.Room, d.Room)
			})

			for k := range result[i].Buildings[j].Rooms {
				slices.SortFunc(result[i].Buildings[j].Rooms[k].Events, func(e, f schema.MazevoEvent) int {
					// sorts alphabetically
					return strings.Compare(*e.EventName, *f.EventName)
				})
			}
		}
	}

	log.Print("Parsed Mazevo!")

	err = utils.WriteJSON(fmt.Sprintf("%s/mazevo.json", outDir), result)
	if err != nil {
		log.Panic(err)
	}
}
