package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func mazevoTestDirs(t *testing.T) (string, string) {
	t.Helper()
	inDir, outDir := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(inDir, "mazevo"), 0o755); err != nil {
		t.Fatal(err)
	}
	return inDir, outDir
}

func writeMazevoInput(t *testing.T, inDir, name, data string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(inDir, "mazevo", name), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mazevoDate(date, building, room string, events ...schema.MazevoEvent) schema.MultiBuildingEvents[schema.MazevoEvent] {
	return schema.MultiBuildingEvents[schema.MazevoEvent]{
		Date: date,
		Buildings: []schema.SingleBuildingEvents[schema.MazevoEvent]{
			{Building: building, Rooms: []schema.RoomEvents[schema.MazevoEvent]{
				{Room: room, Events: events},
			}},
		},
	}
}

func checkMazevoOutput(t *testing.T, outDir string, want []schema.MultiBuildingEvents[schema.MazevoEvent]) {
	t.Helper()
	got := readJSONFile[[]schema.MultiBuildingEvents[schema.MazevoEvent]](t, filepath.Join(outDir, "mazevo.json"))
	// Map iteration does not define the order of dates, buildings, or rooms.
	// Keep event order intact so missing, duplicate, or reordered events are detected.
	if diff := cmp.Diff(want, got,
		cmpopts.SortSlices(func(a, b schema.MultiBuildingEvents[schema.MazevoEvent]) bool { return a.Date < b.Date }),
		cmpopts.SortSlices(func(a, b schema.SingleBuildingEvents[schema.MazevoEvent]) bool { return a.Building < b.Building }),
		cmpopts.SortSlices(func(a, b schema.RoomEvents[schema.MazevoEvent]) bool { return a.Room < b.Room }),
	); diff != "" {
		t.Fatalf("unexpected Mazevo output (-want +got):\n%s", diff)
	}
}

// TestParseMazevo_CompleteBooking verifies all event fields and their placement under the correct date, building, and room.
func TestParseMazevo_CompleteBooking(t *testing.T) {
	t.Parallel()
	inDir, outDir := mazevoTestDirs(t)
	writeMazevoInput(t, inDir, "events.json", `{"bookings":[{
		"buildingDescription":"Student Union", "roomDescription":"SU 2.602",
		"eventName":"Club Meeting", "organizationName":"Nebula Labs", "contactName":"Alex",
		"setupMinutes":15, "dateTimeStart":"2026-10-15T10:00:00-05:00",
		"dateTimeEnd":"2026-10-15T11:00:00-05:00", "teardownMinutes":7.5,
		"statusDescription":"Confirmed", "statusColor":"#00ff00"
	}]}`)
	setup, teardown := 15.0, 7.5
	want := schema.MazevoEvent{
		EventName: strPtr("Club Meeting"), OrganizationName: strPtr("Nebula Labs"),
		ContactName: strPtr("Alex"), SetupMinutes: &setup,
		DateTimeStart: strPtr("2026-10-15T10:00:00-05:00"),
		DateTimeEnd:   strPtr("2026-10-15T11:00:00-05:00"), TeardownMinutes: &teardown,
		StatusDescription: strPtr("Confirmed"), StatusColor: strPtr("#00ff00"),
	}
	ParseMazevo(inDir, outDir)
	checkMazevoOutput(t, outDir, []schema.MultiBuildingEvents[schema.MazevoEvent]{mazevoDate("2026-10-15", "SU", "2.602", want)})
}

// TestParseMazevo_NormalizesLocations checks building renames, whitespace trimming, and exact room prefix removal.
func TestParseMazevo_NormalizesLocations(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, building, room, wantBuilding, wantRoom string }{
		{"student_union", " Student Union ", "SU 2.602", "SU", "2.602"},
		{"student_services", "Student Services Addition (SSA)", "SSA 12.120", "SSA", "12.120"},
		{"unknown_building", " Library ", "Lobby", "Library", "Lobby"},
		{"already_abbreviated", "SU", "2.602", "SU", "2.602"},
		{"prefix_requires_space", "Student Union", "SU2.602", "SU", "SU2.602"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inDir, outDir := mazevoTestDirs(t)
			writeJSONFile(t, filepath.Join(inDir, "mazevo", "events.json"), SourceData{Bookings: []map[string]interface{}{
				{"dateTimeStart": "2026-10-15T10:00:00", "buildingDescription": tt.building, "roomDescription": tt.room},
			}})
			ParseMazevo(inDir, outDir)
			checkMazevoOutput(t, outDir, []schema.MultiBuildingEvents[schema.MazevoEvent]{
				mazevoDate("2026-10-15", tt.wantBuilding, tt.wantRoom, schema.MazevoEvent{DateTimeStart: strPtr("2026-10-15T10:00:00")}),
			})
		})
	}
}

// TestParseMazevo_GroupsAcrossFiles verifies grouping across input files and confirms that subdirectories are skipped.
func TestParseMazevo_GroupsAcrossFiles(t *testing.T) {
	t.Parallel()
	inDir, outDir := mazevoTestDirs(t)
	writeMazevoInput(t, inDir, "01.json", `{"bookings":[
		{"dateTimeStart":"2026-10-15", "buildingDescription":"Student Union", "roomDescription":"SU 1", "eventName":"First"},
		{"dateTimeStart":"2026-10-15", "buildingDescription":"SU", "roomDescription":"2", "eventName":"Other room"},
		{"dateTimeStart":"2026-10-15", "buildingDescription":"Library", "roomDescription":"1", "eventName":"Other building"}
	]}`)
	writeMazevoInput(t, inDir, "02.json", `{"bookings":[
		{"dateTimeStart":"2026-10-15", "buildingDescription":"SU", "roomDescription":"1", "eventName":"Second"},
		{"dateTimeStart":"2026-10-16", "buildingDescription":"SU", "roomDescription":"1", "eventName":"Other date"}
	]}`)
	// An invalid JSON file inside a subdirectory must not be read.
	nested := filepath.Join(inDir, "mazevo", "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "invalid.json"), []byte("invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	event := func(name, date string) schema.MazevoEvent {
		return schema.MazevoEvent{EventName: strPtr(name), DateTimeStart: strPtr(date)}
	}
	first := mazevoDate("2026-10-15", "SU", "1", event("First", "2026-10-15"), event("Second", "2026-10-15"))
	first.Buildings[0].Rooms = append(first.Buildings[0].Rooms, schema.RoomEvents[schema.MazevoEvent]{
		Room: "2", Events: []schema.MazevoEvent{event("Other room", "2026-10-15")},
	})
	first.Buildings = append(first.Buildings, mazevoDate("2026-10-15", "Library", "1", event("Other building", "2026-10-15")).Buildings[0])
	ParseMazevo(inDir, outDir)
	checkMazevoOutput(t, outDir, []schema.MultiBuildingEvents[schema.MazevoEvent]{
		first, mazevoDate("2026-10-16", "SU", "1", event("Other date", "2026-10-16")),
	})
}

// TestParseMazevo_SkipsInvalidRequiredFields checks that invalid required fields cause bookings to be skipped while valid bookings are retained.
func TestParseMazevo_SkipsInvalidRequiredFields(t *testing.T) {
	t.Parallel()
	for _, field := range []string{"dateTimeStart", "buildingDescription", "roomDescription"} {
		values := []struct {
			name  string
			value any
		}{
			{"missing", nil}, {"null", nil}, {"wrong_type", 42},
		}
		if field != "dateTimeStart" {
			values = append(values, struct {
				name  string
				value any
			}{"empty", ""})
		}
		for _, tt := range values {
			t.Run(field+"/"+tt.name, func(t *testing.T) {
				t.Parallel()
				inDir, outDir := mazevoTestDirs(t)
				invalid := map[string]interface{}{"dateTimeStart": "2026-10-15", "buildingDescription": "SU", "roomDescription": "1"}
				if tt.name == "missing" {
					delete(invalid, field)
				} else {
					invalid[field] = tt.value
				}
				valid := map[string]interface{}{"dateTimeStart": "2026-10-15", "buildingDescription": "SU", "roomDescription": "1"}
				writeJSONFile(t, filepath.Join(inDir, "mazevo", "events.json"), SourceData{Bookings: []map[string]interface{}{invalid, valid}})
				ParseMazevo(inDir, outDir)
				checkMazevoOutput(t, outDir, []schema.MultiBuildingEvents[schema.MazevoEvent]{
					mazevoDate("2026-10-15", "SU", "1", schema.MazevoEvent{DateTimeStart: strPtr("2026-10-15")}),
				})
			})
		}
	}
}

// TestParseMazevo_OptionalFields verifies that missing, null, or wrong-type optional fields become nil while valid zero values are preserved.
func TestParseMazevo_OptionalFields(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"missing", "null", "wrong_type", "zero_values"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			inDir, outDir := mazevoTestDirs(t)
			booking := map[string]interface{}{"dateTimeStart": "2026-10-15", "buildingDescription": "SU", "roomDescription": "1"}
			want := schema.MazevoEvent{DateTimeStart: strPtr("2026-10-15")}
			if name != "missing" {
				for _, field := range []string{"eventName", "organizationName", "contactName", "dateTimeEnd", "statusDescription", "statusColor"} {
					var value any
					if name == "wrong_type" {
						value = 42
					} else if name == "zero_values" {
						value = ""
					}
					booking[field] = value
				}
				for _, field := range []string{"setupMinutes", "teardownMinutes"} {
					var value any
					if name == "wrong_type" {
						value = "15"
					} else if name == "zero_values" {
						value = 0
					}
					booking[field] = value
				}
			}
			if name == "zero_values" {
				zero := 0.0
				want.EventName, want.OrganizationName, want.ContactName = strPtr(""), strPtr(""), strPtr("")
				want.DateTimeEnd, want.StatusDescription, want.StatusColor = strPtr(""), strPtr(""), strPtr("")
				want.SetupMinutes, want.TeardownMinutes = &zero, &zero
			}
			writeJSONFile(t, filepath.Join(inDir, "mazevo", "events.json"), SourceData{Bookings: []map[string]interface{}{booking}})
			ParseMazevo(inDir, outDir)
			checkMazevoOutput(t, outDir, []schema.MultiBuildingEvents[schema.MazevoEvent]{mazevoDate("2026-10-15", "SU", "1", want)})
		})
	}
}

// TestParseMazevo_EmptyData checks that input with no accepted bookings produces JSON null.
func TestParseMazevo_EmptyData(t *testing.T) {
	t.Parallel()
	for name, input := range map[string]string{
		"empty_directory": "", "missing_bookings": `{}`, "null_document": `null`,
		"null_bookings": `{"bookings":null}`, "empty_bookings": `{"bookings":[]}`,
		"all_skipped": `{"bookings":[{},null]}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			inDir, outDir := mazevoTestDirs(t)
			if input != "" {
				writeMazevoInput(t, inDir, "events.json", input)
			}
			ParseMazevo(inDir, outDir)
			checkMazevoOutput(t, outDir, nil)
		})
	}
}

// TestParseMazevo_MissingInputDirectory verifies that a missing input directory returns without creating output.
func TestParseMazevo_MissingInputDirectory(t *testing.T) {
	t.Parallel()
	inDir, outDir := t.TempDir(), t.TempDir()
	ParseMazevo(inDir, outDir)
	if _, err := os.Stat(filepath.Join(outDir, "mazevo.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no output, got stat error: %v", err)
	}
}

// TestParseMazevo_InvalidInputPanics documents the existing panics for malformed JSON, incompatible booking structures, and short start dates.
func TestParseMazevo_InvalidInputPanics(t *testing.T) {
	t.Parallel()
	for name, input := range map[string]string{
		"empty_file": "", "malformed_json": `{"bookings":`,
		"wrong_bookings_type": `{"bookings":{}}`, "wrong_booking_type": `{"bookings":[42]}`,
		"empty_start": `{"bookings":[{"dateTimeStart":""}]}`,
		"short_start": `{"bookings":[{"dateTimeStart":"2026"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			inDir, outDir := mazevoTestDirs(t)
			writeMazevoInput(t, inDir, "events.json", input)
			defer func() {
				if recover() == nil {
					t.Error("expected ParseMazevo to panic")
				}
				if _, err := os.Stat(filepath.Join(outDir, "mazevo.json")); !os.IsNotExist(err) {
					t.Errorf("expected no output after invalid input, got stat error: %v", err)
				}
			}()
			ParseMazevo(inDir, outDir)
		})
	}
}

// TestParseMazevo_FileErrors verifies that input-read and output-write failures cause panics.
func TestParseMazevo_FileErrors(t *testing.T) {
	t.Parallel()
	t.Run("unreadable_input", func(t *testing.T) {
		t.Parallel()
		inDir, outDir := mazevoTestDirs(t)
		if err := os.Symlink(filepath.Join(inDir, "missing.json"), filepath.Join(inDir, "mazevo", "broken.json")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		defer func() {
			if _, ok := recover().(*os.PathError); !ok {
				t.Error("expected a file-read panic containing *os.PathError")
			}
		}()
		ParseMazevo(inDir, outDir)
	})
	t.Run("output_path_is_directory", func(t *testing.T) {
		t.Parallel()
		inDir, outDir := mazevoTestDirs(t)
		if err := os.Mkdir(filepath.Join(outDir, "mazevo.json"), 0o755); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if recover() == nil {
				t.Error("expected an output-write panic")
			}
		}()
		ParseMazevo(inDir, outDir)
	})
}
