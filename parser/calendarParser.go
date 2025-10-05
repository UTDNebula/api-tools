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