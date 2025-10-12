package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

// Some events have only the building name, not the abbreviation
// Maps building names to their abbreviations
var buildingAbbreviations = map[string]string{
	"Activity Center":                              "AB",
	"Activity Center Bookstore":                    "ACB",
	"Administration":                               "AD",
	"Edith and Peter O’Donnell Jr. Athenaeum":      "APC",
	"Edith O'Donnell Arts and Technology Building": "ATC",
	"Lloyd V. Berkner Hall":                        "BE",
	"Bioengineering and Sciences Building":         "BSB",
	"Classroom Building":                           "CB",
	"Callier Center Richardson":                    "CR",
	"Callier Center Addition":                      "CRA",
	"Davidson-Gundy Alumni Center":                 "DGA",
	"Dining Hall West":                             "DHW",
	"Engineering and Computer Science North":       "ECSN",
	"Engineering and Computer Science South":       "ECSS",
	"Engineering and Computer Science West":        "ECSW",
	"Energy Plant":                                 "EP",
	"Founders Annex":                               "FA",
	"Facilities Management":                        "FM",
	"Founders North":                               "FN",
	"Founders Building":                            "FO",
	"Cecil H. Green Hall":                          "GR",
	"Karl Hoblitzelle Hall":                        "HH",
	"Erik Jonsson Academic Center":                 "JO",
	"Naveen Jindal School of Management":           "JSOM",
	"Eugene McDermott Library":                     "MC",
	"Modular Lab 1":                                "ML1",
	"Modular Lab 2":                                "ML2",
	"North Office Building":                        "NB",
	"North Lab":                                    "NL",
	"Police":                                       "PD",
	"Physics Annex":                                "PHA",
	"Physics Building":                             "PHY",
	"Natural Science and Engineering Research Lab": "RL",
	"Research and Operations Center":               "ROC",
	"Research and Operations Center West":          "ROW",
	"Service Building":                             "SB",
	"Sciences Building":                            "SCI",
	"Safety and Grounds":                           "SG",
	"Student Learning Center":                      "SLC",
	"Student Services Building Addition":           "SSA",
	"Student Services Building":                    "SSB",
	"Student Union":                                "SU",
	"Student Union Food Court":                     "SUFC",
	"Synergy Park North":                           "SPN",
	"Synergy Park North 2":                         "SP2",
	"University Theater":                           "TH",
	"Visitor Center":                               "VC",
	"Waterview Science and Technology Center":      "WSTC",
	"Andromeda Hall & University Housing Office":   "RHA",
	"Capella Hall":                                 "RHC",
	"Helix Hall":                                   "RHH",
	"Sirius Hall":                                  "RHS",
	"Vega Hall":                                    "RHV",
	"Recreation Center West":                       "RCW",
	"SP/N Gallery":                                 "SP2",
}

// Valid building abreviations for checking
var validAbbreviations []string = []string{
	"AB",
	"ACB",
	"AD",
	"APC",
	"ATC",
	"BE",
	"BSB",
	"CB",
	"CR",
	"CRA",
	"DGA",
	"DHW",
	"ECSN",
	"ECSS",
	"ECSW",
	"EP",
	"FA",
	"FM",
	"FN",
	"FO",
	"GR",
	"HH",
	"JO",
	"JSOM",
	"MC",
	"ML1",
	"ML2",
	"NB",
	"NL",
	"PD",
	"PHA",
	"PHY",
	"RL",
	"ROC",
	"ROW",
	"SB",
	"SCI",
	"SG",
	"SLC",
	"SSA",
	"SSB",
	"SU",
	"SUFC",
	"SPN",
	"SP2",
	"TH",
	"VC",
	"WSTC",
	"RHA",
	"RHC",
	"RHH",
	"RHS",
	"RHV",
	"RCW",
}

// Some events refer to the room name instead of their number
// It's very likely that there are other named rooms with room numbers not added yet
// Maps room names to room number
var roomNumbers = map[string]string {
	"Artemis I":                                    "2.905A",
	"Artemis II":                                   "2.905B",
	"Main Gym":                                     "1.2",
	"Auxiliary Gym":                                "1.318",
	"Axxess Atrium":                                "1.100",
	"Ballroom A":                                   "1.102A",
	"Ballroom B":                                   "1.102B",
	"Ballroom C":                                   "1.102C",
	"AHT Gallery":                                  "3.102",
	"SP/N Gallery":                                 "11.150",
	"Galaxy Rooms":                                 "2.602",
	"ATC Auditorium":                               "1.102",
	"ATC Lecture Hall":                             "l.l02",
	"TI Auditorium":                                "2.102",
	"SSA Auditorium":                               "13.330",
	"Clark Auditorium":                             "1.315",
	"ATC Lobby":                                    "1.700",
}

func ParseCalendar(inDir string, outDir string) {
	
	calendarFile, err := os.ReadFile(inDir + "/eventScraped.json")
	if err != nil {
		panic(err)
	}
	
	var allEvents []schema.Event

	err = json.Unmarshal(calendarFile, &allEvents)
	if err != nil {
		panic(err)
	}

	multiBuildingMap := make(map[string]map[string]map[string][]schema.Event)

	for _, event := range(allEvents) {

		// Get date
		dateTime := event.StartTime
		dateTimeString := dateTime.String()
		date := dateTimeString[:10]

		// Get building and room
		location := utils.ConvertFromInterface[string](event.Location)

		// Regexp to match building abbreviations and room numbers
		buildingRegexp := regexp.MustCompile(`[A-Z]{2,4}`)
		roomRegexp := regexp.MustCompile(`([0-9]{1,2}\.[0-9]{3})([A-Z])?`)

		building := buildingRegexp.FindString(*location)
		room := roomRegexp.FindString(*location)

		// buildingRegexp might capture something that isn't a valid building abbreviation (e.g., UTD)
		if checkBuilding := slices.Contains(validAbbreviations, building); !checkBuilding {
			building = ""
		}
		
		lowercaseLocation := strings.ToLower(*location)
		// If location doesn't have building abbreviation, check for the full building name
		if building == "" {
			for key := range buildingAbbreviations {
				if strings.Contains(lowercaseLocation, strings.ToLower(key)) {
					building = buildingAbbreviations[key]
				}
			}
		}
		
		// If location doesn't have room number, check for room names
		if room == "" {
			for key := range roomNumbers {
				if strings.Contains(lowercaseLocation, strings.ToLower(key)) {
					room = roomNumbers[key]
				}
			}
		}

		// If building is still empty string, then location was initally an empty string
		// or was a place off campus
		if building == "" {
			building = "Other"
		}

		// If room is still empty string, then location was initally an empty string, or
		// the room had no equivalent room number, or was a place off campus
		if room == "" {
			room = "Other"
		}

		if _, exists := multiBuildingMap[date]; !exists {
			multiBuildingMap[date] = make(map[string]map[string][]schema.Event)
		}

		if _, exists := multiBuildingMap[date][building]; !exists {
			multiBuildingMap[date][building] = make(map[string][]schema.Event)
		}

		multiBuildingMap[date][building][room] = append(multiBuildingMap[date][building][room], event)
	}

	var result []schema.MultiBuildingEvents[schema.Event]

	for date, buildings := range multiBuildingMap {
		var singleBuildings []schema.SingleBuildingEvents[schema.Event]
		for building, rooms := range buildings {
			var roomEvents []schema.RoomEvents[schema.Event]
			for room, events := range rooms {
				roomEvents = append(roomEvents, schema.RoomEvents[schema.Event]{
					Room:   room,
					Events: events,
				})
			}

			singleBuildings = append(singleBuildings, schema.SingleBuildingEvents[schema.Event]{
				Building: building,
				Rooms:    roomEvents,
			})
		}

		result = append(result, schema.MultiBuildingEvents[schema.Event]{
			Date:      date,
			Buildings: singleBuildings,
		})
	}
	
	log.Print("Parsed Calendar!")

	utils.WriteJSON(fmt.Sprintf("%s/events.json", outDir), result)
}