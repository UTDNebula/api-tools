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

// InputData describes the raw Astra export payload containing fields metadata and row values.
type InputData struct {
	Fields string          `json:"fields"`
	Data   [][]interface{} `json:"data"`
}

// ParseAstra reads Astra scrape output and produces structured multi-building event JSON files.
func ParseAstra(inDir string, outDir string) {

	astraFile, err := os.ReadFile(inDir + "/astraScraped.json")
	if err != nil {
		panic(err)
	}

	var rawData map[string]InputData
	err = json.Unmarshal(astraFile, &rawData)
	if err != nil {
		panic(err)
	}

	var result []schema.MultiBuildingEvents[schema.AstraEvent]

	for date, data := range rawData {
		fieldMap := mapFields(data.Fields)
		buildingsMap := make(map[string]map[string][]schema.AstraEvent)

		for _, record := range data.Data {
			building := getString(record, fieldMap["BuildingCode"])
			room := getString(record, fieldMap["RoomNumber"])
			event := schema.AstraEvent{
				ActivityName:        getString(record, fieldMap["ActivityName"]),
				MeetingType:         getString(record, fieldMap["MeetingType"]),
				StartDate:           getString(record, fieldMap["StartDate"]),
				EndDate:             getString(record, fieldMap["EndDate"]),
				CurrentState:        getString(record, fieldMap["CurrentState"]),
				NotAllowedUsageMask: getInt(record, fieldMap["NotAllowedUsageMask"]),
				UsageColor:          getString(record, fieldMap["UsageColor"]),
				Capacity:            getInt(record, fieldMap["Capacity"]),
			}

			if building == nil || room == nil || *(building) == "" || *(room) == "" {
				continue
			}

			if _, exists := buildingsMap[*building]; !exists {
				buildingsMap[*building] = make(map[string][]schema.AstraEvent)
			}
			buildingsMap[*building][*room] = append(buildingsMap[*building][*room], event)
		}

		var buildings []schema.SingleBuildingEvents[schema.AstraEvent]
		for buildingCode, rooms := range buildingsMap {
			var roomList []schema.RoomEvents[schema.AstraEvent]
			for roomNumber, events := range rooms {
				roomList = append(roomList, schema.RoomEvents[schema.AstraEvent]{Room: roomNumber, Events: events})
			}
			buildings = append(buildings, schema.SingleBuildingEvents[schema.AstraEvent]{Building: buildingCode, Rooms: roomList})
		}
		data := schema.MultiBuildingEvents[schema.AstraEvent]{
			Date:      date,
			Buildings: buildings,
		}
		result = append(result, data)
	}

	log.Print("Parsed Astra!")

	utils.WriteJSON(fmt.Sprintf("%s/astra.json", outDir), result)
}

func mapFields(fields string) map[string]int {
	fieldNames := map[string]int{}
	fieldList := strings.Split(fields, ",")
	for i, name := range fieldList {
		fieldNames[name] = i
	}
	return fieldNames
}

func getString(record []interface{}, place int) *string {
	val := record[place]
	if val == nil {
		return nil
	}
	s := val.(string)
	return &s
}

func getInt(record []interface{}, place int) *float64 {
	val := record[place]
	if val == nil {
		return nil
	}
	i := val.(float64)
	return &i
}
