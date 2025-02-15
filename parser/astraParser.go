package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

type InputData struct {
	Fields string          `json:"fields"`
	Data   [][]interface{} `json:"data"`
}

func ParseAstra(inDir string, outDir string) {

	astraFile, err := os.ReadFile(inDir + "/astraReservations.json")
	if err != nil {
		panic(err)
	}

	var rawData map[string]InputData
	err = json.Unmarshal(astraFile, &rawData)
	if err != nil {
		panic(err)
	}

	result := make(map[string]schema.MultiBuildingEvents[schema.AstraEvent])

	for date, data := range rawData {
		fieldMap := mapFields(data.Fields)
		buildingsMap := make(map[string]map[string][]schema.AstraEvent)

		for _, record := range data.Data {
			event := schema.AstraEvent{
				ActivityName:        getString(record, fieldMap["ActivityName"]),
				MeetingType:         getString(record, fieldMap["MeetingType"]),
				StartDate:           getString(record, fieldMap["StartDate"]),
				EndDate:             getString(record, fieldMap["EndDate"]),
				BuildingCode:        getString(record, fieldMap["BuildingCode"]),
				RoomNumber:          getString(record, fieldMap["RoomNumber"]),
				CurrentState:        getString(record, fieldMap["CurrentState"]),
				NotAllowedUsageMask: getInt(record, fieldMap["NotAllowedUsageMask"]),
				UsageColor:          getString(record, fieldMap["UsageColor"]),
				Capacity:            getInt(record, fieldMap["Capacity"]),
			}

			if event.BuildingCode == nil || event.RoomNumber == nil || *(event.BuildingCode) == "" || *(event.RoomNumber) == "" {
				continue
			}

			if _, exists := buildingsMap[*(event.BuildingCode)]; !exists {
				buildingsMap[*(event.BuildingCode)] = make(map[string][]schema.AstraEvent)
			}
			buildingsMap[*(event.BuildingCode)][*(event.RoomNumber)] = append(buildingsMap[*(event.BuildingCode)][*(event.RoomNumber)], event)
		}

		var buildings []schema.SingleBuildingEvents[schema.AstraEvent]
		for buildingCode, rooms := range buildingsMap {
			var roomList []schema.RoomEvents[schema.AstraEvent]
			for roomNumber, events := range rooms {
				roomList = append(roomList, schema.RoomEvents[schema.AstraEvent]{Room: roomNumber, Events: events})
			}
			buildings = append(buildings, schema.SingleBuildingEvents[schema.AstraEvent]{Building: buildingCode, Rooms: roomList})
		}
		result[date] = schema.MultiBuildingEvents[schema.AstraEvent]{Buildings: buildings}
	}

	utils.WriteJSON(fmt.Sprintf("%s/astraParsed.json", outDir), result)
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
