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
	"time"

	"github.com/UTDNebula/api-tools/scrapers"
	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

// Some events have only the building name, not the abbreviation
// Maps building names to their abbreviations
var DefaultBuildings = map[string]string{
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
	"University Theatre":                           "TH",
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
var DefaultValid []string = []string{
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
	buildingAbbreviations, validAbbreviations := getLocationAbbreviations(inDir)

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

		// If building is still empty string or invalid abbreviation, then location was initally an empty string
		// or location was a place off campus
		if building == "" || !isValidBuilding {
			building = "Other"
		}

		// If room is still empty string, then location was initally an empty string, or
		// location did not include a room, or location was a place off campus
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

// getAbbreviations dynamically retrieves the all of the locations abbreviations
func getLocationAbbreviations(inDir string) (map[string]string, []string) {
	// Get the locations from the map scraper
	var mapFile []byte

	mapFile, err := os.ReadFile(inDir + "/mapLocations.json")
	if err != nil {
		if os.IsNotExist(err) {
			// Scrape the data if the it doesn't exist yet and then get the map file
			scrapers.ScrapeMapLocations(inDir)
			time.Sleep(2 * time.Second)
			ParseMapLocations(inDir, inDir)
			time.Sleep(2 * time.Second)

			// If fail to get the locations again, it's not because location is unscraped
			mapFile, err = os.ReadFile(inDir + "/mapLocations.json")
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}
	var locations []schema.MapBuilding
	if err = json.Unmarshal(mapFile, &locations); err != nil {
		panic(err)
	}

	// Process the abbreviations
	buildingsAbbreviations := make(map[string]string, 0)
	validAbbreviations := make([]string, 0)

	for _, location := range locations {
		// Trim the following acronym in the name
		trimmedName := strings.Split(*location.Name, " (")[0]
		// Fallback on the locations that have no acronyms
		abbreviation := ""
		if location.Acronym != nil {
			abbreviation = *location.Acronym
		}

		buildingsAbbreviations[trimmedName] = abbreviation
		validAbbreviations = append(validAbbreviations, abbreviation)
	}

	return buildingsAbbreviations, validAbbreviations
}
