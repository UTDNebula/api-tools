/*
	This file contains the code for the comet calendar events parser.
*/

package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/UTDNebula/api-tools/scrapers"
	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

// Rename the building and rooms to match the other sources of data
type standardizedBuilding struct {
	building string
	rooms    map[string]string
}

// TODO: Find the better way to avoid hardcoding this
var standardizedMap = map[string]standardizedBuilding{
	"ATC": {
		building: "ATC",
		rooms: map[string]string{
			"ATC Auditorium": "Auditorium",
			"Lecture Hall":   "Auditorium",
			"1.102":          "Auditorium",
		},
	},
	"ECSN": {
		building: "ECSN",
		rooms: map[string]string{
			"1st floor ECSN atrium": "1st Floor ECSN Atrium",
		},
	},
	"FO": {
		building: "FO",
		rooms: map[string]string{
			"Founders 2nd Floor Atrium": "2nd Floor Atrium",
			"2nd floor atrium":          "2nd Floor Atrium",
		},
	},
	"SU": {
		building: "Student Union (SU)",
		rooms: map[string]string{
			"1st Floor":             "First Floor",
			"2nd Floor":             "Second Floor",
			"2.602":                 "Galaxy  Room  (A, B, & C)",
			"Galaxy Rooms":          "Galaxy  Room  (A, B, & C)",
			"Galaxy Rooms A, B & C": "Galaxy  Room  (A, B, & C)",
			"Galaxy Rooms A & B":    "Galaxy  Room  (A & B)",
			"Galaxy A":              "Galaxy  Room - A",
			"Artemis Hall I &II":    "Artemis Hall (I & II)",
		},
	},
	"SB": {
		building: "SSB",
		rooms:    map[string]string{},
	},
	"SSA": {
		building: "SSA",
		rooms: map[string]string{
			"12.120":                              "Atrium",
			"Atrium (formely Gaming Wall Lounge)": "Atrium",
			"13.330":                              "Auditorium",
		},
	},
}

// ParseCometCalendar reformats the comet calendar data into uploadable json in Mongo
func ParseCometCalendar(inDir string, outDir string) {

	calendarFile, err := os.ReadFile(inDir + "/cometCalendarScraped.json")
	if err != nil {
		panic(err)
	}

	var allEvents []schema.Event

	err = json.Unmarshal(calendarFile, &allEvents)
	if err != nil {
		panic(err)
	}

	multiBuildingMap := make(map[string]map[string]map[string][]schema.Event)
	// Some events have only the building name, not the abbreviation
	buildingAbbreviations, validAbbreviations, err := getLocationAbbreviations(inDir)
	if err != nil {
		panic(err)
	}

	for _, event := range allEvents {

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
		isValidBuilding := slices.Contains(validAbbreviations, building)

		// If location doesn't have building abbreviation or buildingRegexp captured an invalid abbreviation,
		// check for the full building name
		lowercaseLocation := strings.ToLower(*location)
		if building == "" || !isValidBuilding {
			for key := range buildingAbbreviations {
				if strings.Contains(lowercaseLocation, strings.ToLower(key)) {
					building = buildingAbbreviations[key]
					isValidBuilding = true
				}
			}
		}

		// If location doesn't have room number, check to see if location included a room
		if room == "" && isValidBuilding {
			locationParts := strings.SplitN(*location, ", ", 2)
			if len(locationParts) == 2 {
				room = locationParts[1]
			}
		}

		// If building is still empty string or invalid abbreviation, then location wasn't provided
		if building == "" || !isValidBuilding {
			building = "Other"
		}

		// If room is still empty string, then location wasn't provided, or
		// location did not include a room
		if room == "" {
			room = "Other"
		}

		if _, exists := standardizedMap[building]; exists {
			standardized := standardizedMap[building]
			building = standardized.building
			if _, exists := standardized.rooms[room]; exists {
				room = standardized.rooms[room]
			}
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
					Room:   strings.TrimSpace(room),
					Events: events,
				})
			}

			singleBuildings = append(singleBuildings, schema.SingleBuildingEvents[schema.Event]{
				Building: strings.TrimSpace(building),
				Rooms:    roomEvents,
			})
		}

		result = append(result, schema.MultiBuildingEvents[schema.Event]{
			Date:      date,
			Buildings: singleBuildings,
		})
	}

	log.Print("Parsed Comet Calendar!")

	utils.WriteJSON(fmt.Sprintf("%s/cometCalendar.json", outDir), result)
}

// getLocationAbbreviations dynamically retrieves the all of the locations abbreviations
func getLocationAbbreviations(inDir string) (map[string]string, []string, error) {
	// Get the locations from the map scraper
	var mapFile []byte
	mapFile, err := os.ReadFile(inDir + "/mapLocations.json")
	if err != nil {
		if os.IsNotExist(err) {
			// Force scrape the locations if it doesn't exist. Get the map file again
			scrapers.ScrapeMapLocations(inDir)
			ParseMapLocations(inDir, inDir)

			// If it fails to get the locations again, it's not because location is unscraped
			mapFile, err = os.ReadFile(inDir + "/mapLocations.json")
			if err != nil {
				return nil, nil, err
			}
		} else {
			return nil, nil, err
		}
	}
	var locations []schema.MapBuilding
	if err = json.Unmarshal(mapFile, &locations); err != nil {
		return nil, nil, err
	}

	// Process the abbreviations
	buildingsAbbreviations := make(map[string]string, 0) // Maps building names to their abbreviations
	validAbbreviations := make([]string, 0)              // Valid building abreviations for checking

	for _, location := range locations {
		// Trim the following acronym in the name
		trimmedName := strings.Split(*location.Name, " (")[0]
		// Fallback on the locations that have no acronyms
		var abbreviation string
		if location.Acronym != nil {
			abbreviation = *location.Acronym
		}

		buildingsAbbreviations[trimmedName] = abbreviation
		validAbbreviations = append(validAbbreviations, abbreviation)
	}

	return buildingsAbbreviations, validAbbreviations, nil
}
