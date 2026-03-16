package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/google/go-cmp/cmp"
)

// helper function to read JSON files into Go structs for testing purposes
func readJSONFile[T any](t *testing.T, path string) T {
	t.Helper()

	var result T

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %q: %v", path, err)
	}

	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON from file %q: %v", path, err)
	}

	return result
}

// helper function to write Go structs to JSON files for testing purposes
func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal value to JSON: %v", err)
	}

	if err := os.WriteFile(path, data, 0o777); err != nil {
		t.Fatalf("failed to write file %q: %v", path, err)
	}
}

// helper function to create a pointer to a string to match expected struct field
func strPtr(s string) *string {
	return &s
}

// helper function to create valid schema.Event object for testing purposes
func makeEvent(summary, location string, startTime time.Time) schema.Event {
	return schema.Event{
		Id:        primitive.NewObjectID(),
		Summary:   summary,
		Location:  location,
		StartTime: startTime,
		EndTime:   startTime.Add(time.Hour),
	}
}

// helper function to find a specific date entry in the parser result and fail if it is missing
func findDate(
	t *testing.T,
	result []schema.MultiBuildingEvents[schema.Event],
	date string,
) *schema.MultiBuildingEvents[schema.Event] {
	t.Helper()

	for i := range result {
		if result[i].Date == date {
			return &result[i]
		}
	}

	t.Fatalf("date %q not found in result", date)
	return nil
}

// helper function to find a specific building entry under a date and fail if it is missing
func findBuilding(
	t *testing.T,
	dateEntry *schema.MultiBuildingEvents[schema.Event],
	building string,
) *schema.SingleBuildingEvents[schema.Event] {
	t.Helper()

	for i := range dateEntry.Buildings {
		if dateEntry.Buildings[i].Building == building {
			return &dateEntry.Buildings[i]
		}
	}

	t.Fatalf("building %q not found under date %q", building, dateEntry.Date)
	return nil
}

// helper function to find a specific room entry under a building and fail if it is missing
func findRoom(
	t *testing.T,
	buildingEntry *schema.SingleBuildingEvents[schema.Event],
	room string,
) *schema.RoomEvents[schema.Event] {
	t.Helper()

	for i := range buildingEntry.Rooms {
		if buildingEntry.Rooms[i].Room == room {
			return &buildingEntry.Rooms[i]
		}
	}

	t.Fatalf("room %q not found under building %q", room, buildingEntry.Building)
	return nil
}

// helper function to create a sample set of map locations for testing purposes
func testMapLocations() []schema.MapBuilding {
	return []schema.MapBuilding{
		{
			Name:    strPtr("Engineering and Computer Science South (ECSS)"),
			Acronym: strPtr("ECSS"),
		},
		{
			Name:    strPtr("Jonsson Performance Hall (JO)"),
			Acronym: strPtr("JO"),
		},
	}
}

// tests that getLocationAbbreviations correctly reads the mapLocations.json file and returns the expected building abbreviations and valid abbreviation list
func TestGetLocationAbbreviations_Success(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()
	writeJSONFile(t, filepath.Join(inDir, "mapLocations.json"), testMapLocations())

	buildingAbbreviations, validAbbreviations, err := getLocationAbbreviations(inDir)
	if err != nil {
		t.Fatalf("getLocationAbbreviations returned an error: %v", err)
	}

	if got := buildingAbbreviations["Engineering and Computer Science South"]; got != "ECSS" {
		t.Fatalf("expected Engineering and Computer Science South -> ECSS, got %q", got)
	}

	if got := buildingAbbreviations["Jonsson Performance Hall"]; got != "JO" {
		t.Fatalf("expected Jonsson Performance Hall -> JO, got %q", got)
	}

	if !slices.Contains(validAbbreviations, "ECSS") {
		t.Fatalf("expected validAbbreviations to contain ECSS, got %v", validAbbreviations)
	}

	if !slices.Contains(validAbbreviations, "JO") {
		t.Fatalf("expected validAbbreviations to contain JO, got %v", validAbbreviations)
	}
}

// Tests that if a building has no acronym, an empty string is used as the abbreviation and is included in the validAbbreviations list
func TestGetLocationAbbreviations_NoAcronym(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()

	locations := []schema.MapBuilding{
		{
			Name: strPtr("Unknown Building"),
		},
	}

	writeJSONFile(t, filepath.Join(inDir, "mapLocations.json"), locations)

	buildingAbbreviations, validAbbreviations, err := getLocationAbbreviations(inDir)
	if err != nil {
		t.Fatalf("getLocationAbbreviations returned an error: %v", err)
	}

	if got := buildingAbbreviations["Unknown Building"]; got != "" {
		t.Fatalf("expected empty-string abbreviation for building with no acronym, got %q", got)
	}

	if !slices.Contains(validAbbreviations, "") {
		t.Fatalf("expected validAbbreviations to contain empty string, got %v", validAbbreviations)
	}
}

// Tests that getLocationAbbreviations returns an error when mapLocations.json contains invalid JSON.
func TestGetLocationAbbreviations_InvalidJSON(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(inDir, "mapLocations.json"), []byte("invalid json"), 0o777); err != nil {
		t.Fatalf("failed to write invalid json fixture: %v", err)
	}

	_, _, err := getLocationAbbreviations(inDir)
	if err == nil {
		t.Fatalf("expected error for invalid mapLocations.json, got nil")
	}
}

// Tests that ParseCometCalendar correctly processes a single event and stores it in the expected location in the output JSON structure.
func TestParseCometCalendar_ParsesAbbreviationAndRoom(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()
	outDir := t.TempDir()

	writeJSONFile(t, filepath.Join(inDir, "mapLocations.json"), testMapLocations())

	start := time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC)
	events := []schema.Event{
		makeEvent("Test Event", "ECSS 2.415", start),
	}
	writeJSONFile(t, filepath.Join(inDir, "cometCalendarScraped.json"), events)

	ParseCometCalendar(inDir, outDir)

	result := readJSONFile[[]schema.MultiBuildingEvents[schema.Event]](
		t,
		filepath.Join(outDir, "cometCalendar.json"),
	)

	dateEntry := findDate(t, result, "2026-03-14")
	buildingEntry := findBuilding(t, dateEntry, "ECSS")
	roomEntry := findRoom(t, buildingEntry, "2.415")

	if len(roomEntry.Events) != 1 {
		t.Fatalf("expected 1 event in ECSS/2.415 on 2026-03-14, got %d", len(roomEntry.Events))
	}

	if diff := cmp.Diff(events[0].Summary, roomEntry.Events[0].Summary); diff != "" {
		t.Fatalf("unexpected event stored in room (-want +got);\n%s", diff)
	}
}

// Tests that ParseCometCalendar correctly resolves full building names to their abbreviations.
func TestParseCometCalendar_FallsBackToFullBuildingName(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()
	outDir := t.TempDir()

	writeJSONFile(t, filepath.Join(inDir, "mapLocations.json"), testMapLocations())

	start := time.Date(2026, 3, 14, 10, 0, 0, 0, time.UTC)
	events := []schema.Event{
		makeEvent("Full Building Name Event", "Engineering and Computer Science South 2.415", start),
	}
	writeJSONFile(t, filepath.Join(inDir, "cometCalendarScraped.json"), events)

	ParseCometCalendar(inDir, outDir)

	result := readJSONFile[[]schema.MultiBuildingEvents[schema.Event]](
		t,
		filepath.Join(outDir, "cometCalendar.json"),
	)

	dateEntry := findDate(t, result, "2026-03-14")
	buildingEntry := findBuilding(t, dateEntry, "ECSS")
	roomEntry := findRoom(t, buildingEntry, "2.415")

	if len(roomEntry.Events) != 1 {
		t.Fatalf("expected 1 event in ECSS/2.415 on 2026-03-14, got %d", len(roomEntry.Events))
	}
}

// Test that if an event has a location that does not match any known building name or abbreviation, that it is categorized as "Other"
func TestParseCometCalendar_UsesOtherForUnknownLocation(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()
	outDir := t.TempDir()

	writeJSONFile(t, filepath.Join(inDir, "mapLocations.json"), testMapLocations())

	start := time.Date(2026, 3, 14, 11, 0, 0, 0, time.UTC)
	events := []schema.Event{
		makeEvent("Unknown Location Event", "Off Campus Location", start),
	}
	writeJSONFile(t, filepath.Join(inDir, "cometCalendarScraped.json"), events)

	ParseCometCalendar(inDir, outDir)

	result := readJSONFile[[]schema.MultiBuildingEvents[schema.Event]](
		t,
		filepath.Join(outDir, "cometCalendar.json"),
	)

	dateEntry := findDate(t, result, "2026-03-14")
	buildingEntry := findBuilding(t, dateEntry, "Other")
	roomEntry := findRoom(t, buildingEntry, "Other")

	if len(roomEntry.Events) != 1 {
		t.Fatalf("expected 1 event in Other/Other on 2026-03-14, got %d", len(roomEntry.Events))
	}
}

// Tests that if multiple events occur in the same building/room on the same day, they are grouped together in the JSON output structure
func TestParseCometCalendar_GroupsEventsByDateBuildingRoom(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()
	outDir := t.TempDir()

	writeJSONFile(t, filepath.Join(inDir, "mapLocations.json"), testMapLocations())

	start1 := time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC)
	start2 := time.Date(2026, 3, 14, 13, 0, 0, 0, time.UTC)
	events := []schema.Event{
		makeEvent("First Grouped Event", "ECSS 2.415", start1),
		makeEvent("Second Grouped Event", "ECSS 2.415", start2),
	}
	writeJSONFile(t, filepath.Join(inDir, "cometCalendarScraped.json"), events)

	ParseCometCalendar(inDir, outDir)

	result := readJSONFile[[]schema.MultiBuildingEvents[schema.Event]](
		t,
		filepath.Join(outDir, "cometCalendar.json"),
	)

	dateEntry := findDate(t, result, "2026-03-14")
	buildingEntry := findBuilding(t, dateEntry, "ECSS")
	roomEntry := findRoom(t, buildingEntry, "2.415")

	if len(roomEntry.Events) != 2 {
		t.Fatalf("expected 2 events in ECSS/2.415 on 2026-03-14, got %d", len(roomEntry.Events))
	}
}

// Tests that ParseCometCalendar uses the comma-separated fallback to extract the room when the building is valid and no room was otherwise found.
func TestParseCometCalendar_UsesCommaSeparatedFallbackRoom(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()
	outDir := t.TempDir()

	writeJSONFile(t, filepath.Join(inDir, "mapLocations.json"), testMapLocations())

	start := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	events := []schema.Event{
		makeEvent("Conference Room Event", "ECSS, Conference Room", start),
	}
	writeJSONFile(t, filepath.Join(inDir, "cometCalendarScraped.json"), events)

	ParseCometCalendar(inDir, outDir)

	result := readJSONFile[[]schema.MultiBuildingEvents[schema.Event]](
		t,
		filepath.Join(outDir, "cometCalendar.json"),
	)

	dateEntry := findDate(t, result, "2026-03-14")
	buildingEntry := findBuilding(t, dateEntry, "ECSS")
	roomEntry := findRoom(t, buildingEntry, "Conference Room")

	if len(roomEntry.Events) != 1 {
		t.Fatalf("expected 1 event in the ECSS/Conference Room, got %d", len(roomEntry.Events))
	}
}

// Tests that if an event has an empty location, it is categorized as "Other"
func TestParseCometCalendar_UsesOtherForEmptyLocation(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()
	outDir := t.TempDir()

	writeJSONFile(t, filepath.Join(inDir, "mapLocations.json"), testMapLocations())

	start := time.Date(2026, 3, 14, 14, 0, 0, 0, time.UTC)
	events := []schema.Event{
		makeEvent("Empty Location Event", "", start),
	}
	writeJSONFile(t, filepath.Join(inDir, "cometCalendarScraped.json"), events)

	ParseCometCalendar(inDir, outDir)

	result := readJSONFile[[]schema.MultiBuildingEvents[schema.Event]](
		t,
		filepath.Join(outDir, "cometCalendar.json"),
	)

	dateEntry := findDate(t, result, "2026-03-14")
	buildingEntry := findBuilding(t, dateEntry, "Other")
	roomEntry := findRoom(t, buildingEntry, "Other")

	if len(roomEntry.Events) != 1 {
		t.Fatalf("expected 1 event in Other/Other, got %d", len(roomEntry.Events))
	}
}
