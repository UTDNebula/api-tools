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

func ParseCalendar(inDir string, outDir string) {
	
	calendarFile, err := os.ReadFile(inDir + "/events.json")
	if err != nil {
		panic(err)
	}

	var result []schema.MultiBuildingEvents[schema.Event]
	log.Printf("Test")
}

var buildingAbbreviations = map[string]string{
	"Other Building":                               "",
	"Activity Center":                              "AB",
	"Activity Center Bookstore":                    "ACB",
	"Administration":                               "AD",
	"Edith and Peter O'Donnell Jr. Athenaeum":      "APC",
	"Lloyd V. Berkner Hall":                        "BE",
	"Bioengineering and Sciences Building":         "BSB",
	"Classroom Building":                           "CB",
	"Callier Center Richardson":                    "CR",
	"Callier Center Addition":                      "CRA",
	"Davidson-Gundy Alumni Center":                 "DGA",
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
}
