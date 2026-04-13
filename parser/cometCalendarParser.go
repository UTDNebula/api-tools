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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Regexp to match building abbreviations and room numbers
var (
	buildingRegexp = regexp.MustCompile(`[A-Z]{2,4}`)
	roomRegexp     = regexp.MustCompile(`([0-9]{1,2}\.[0-9]{3})([A-Z])?`)
)

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

		normalizedBuilding := map[string]string{
			"SB": "SSB",
			"SU": "Student Union (SU)",
		}
		if _, ok := normalizedBuilding[building]; ok {
			building = normalizedBuilding[building]
		}

		// If location doesn't have room number, check to see if location included a room
		if room == "" && isValidBuilding {
			locationParts := strings.SplitN(*location, ", ", 2)
			if len(locationParts) == 2 {
				room = locationParts[1]
			}
		}
		room = normalizeRoom(building, room)

		// If building is still empty string or invalid abbreviation, then location wasn't provided
		if building == "" || !isValidBuilding {
			building = "Other"
		}

		// If room is still empty string, then location wasn't provided, or
		// location did not include a room, or the room is invalid
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

// normalizeRoom normalizes the room into set of rooms to match rooms from other event-related data source.
// This is still a little flawed but it can handle 95% of the cases.
func normalizeRoom(building string, room string) string {
	validTokens := []string{
		"first", "second", "third", "floor",
		"galaxy", "a", "b", "c",
		"artemis", "i", "ii", "lecture",
		"atrium", "auditorium", "commons", "lobby",
	}
	tokenMap := map[string]string{
		"1st": "first",
		"2nd": "second",
		"3rd": "third",
	}
	normalizedMap := map[string]string{
		"galaxy a":                 "Galaxy  Room - A",
		"galaxy b":                 "Galaxy  Room - B",
		"galaxy c":                 "Galaxy  Room - C",
		"galaxy a b":               "Galaxy  Room  (A & B)",
		"galaxy b c":               "Galaxy  Room  (B & C)",
		"galaxy a b c":             "Galaxy  Room  (A, B, & C)",
		"galaxy":                   "Galaxy  Room  (A, B, & C)",
		"Student Union (SU) 2.602": "Galaxy  Room  (A, B, & C)",
		"artemis i":                "Artermis Hall I",
		"artemis ii":               "Artemis Hall II",
		"artemis i ii":             "Artemis Hall (I & II)",
		"lecture":                  "Auditorium",
		"SSA 12.120":               "Atrium",
		"SSA 13.330":               "Auditorium",
	}

	if roomRegexp.MatchString(room) {
		// Numeric room, convert to the normalized non-number room if possible
		if _, ok := normalizedMap[building+" "+room]; ok {
			return normalizedMap[building+" "+room]
		}
		return room
	}

	// Non-number room
	room = strings.ToLower(strings.Split(room, ", ")[0])
	tokens := strings.Split(room, " ")
	normalizedTokens := []string{}
	for _, token := range tokens {
		trimmedToken := strings.Trim(token, "!@#$%&()-.,;")
		if _, ok := tokenMap[trimmedToken]; ok {
			trimmedToken = tokenMap[trimmedToken]
		}
		if slices.Contains(validTokens, trimmedToken) {
			normalizedTokens = append(normalizedTokens, trimmedToken)
		}
	}

	normalizedRoom := strings.Join(normalizedTokens, " ")
	if _, ok := normalizedMap[normalizedRoom]; ok {
		return normalizedMap[normalizedRoom]
	}
	return cases.Title(language.English).String(normalizedRoom)
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
	buildingsAbbreviations := make(map[string]string, 0)
	validAbbreviations := make([]string, 0)

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
