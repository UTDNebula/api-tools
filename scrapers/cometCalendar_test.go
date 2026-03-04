package scrapers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCallAndUnmarshalFromURL_Success(t *testing.T) {
	t.Parallel()

	payload, err := os.ReadFile(filepath.Join("testdata", "cometCalendar", "page0.json"))
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("days"); got != "365" {
			t.Errorf("expected query days=365, got %q", got)
		}
		if got := r.URL.Query().Get("pp"); got != "100" {
			t.Errorf("expected query pp=100, got %q", got)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("expected query page=2, got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("expected Accept header application/json, got %q", got)
		}
		if got := r.Header.Get("Content-type"); got != "application/json" {
			t.Errorf("expected Content-type header application/json, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(payload); err != nil {
			t.Errorf("failed to write fixture response: %v", err)
		}
	}))
	defer server.Close()

	client := http.Client{Timeout: 2 * time.Second}
	var calendarData APICalendarResponse

	if err := callAndUnmarshalFromURL(&client, server.URL, 2, &calendarData); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := calendarData.Page["total"]; got != 3 {
		t.Fatalf("expected page.total=3, got %d", got)
	}

	if len(calendarData.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(calendarData.Events))
	}

	event := calendarData.Events[0].Event
	if event.Title != "Nebula Testing Workshop" {
		t.Errorf("expected title %q, got %q", "Nebula Testing Workshop", event.Title)
	}
	if event.Custom_fields.Contact_information_email != "team@utdnebula.com" {
		t.Errorf("expected contact email %q, got %q", "team@utdnebula.com", event.Custom_fields.Contact_information_email)
	}
}

func TestCallAndUnmarshalFromURL_EmptyPayload(t *testing.T) {
	t.Parallel()

	payload, err := os.ReadFile(filepath.Join("testdata", "cometCalendar", "page-empty.json"))
	if err != nil {
		t.Fatalf("failed to load empty fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(payload); err != nil {
			t.Fatalf("failed to write fixture response: %v", err)
		}
	}))
	defer server.Close()

	client := http.Client{Timeout: 2 * time.Second}
	var calendarData APICalendarResponse

	if err := callAndUnmarshalFromURL(&client, server.URL, 1, &calendarData); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := len(calendarData.Events); got != 0 {
		t.Fatalf("expected 0 events, got %d", got)
	}
	if got := calendarData.Page["total"]; got != 0 {
		t.Fatalf("expected page.total=0, got %d", got)
	}
}

func TestCallAndUnmarshalFromURL_Non200(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := http.Client{Timeout: 2 * time.Second}
	var calendarData APICalendarResponse

	if err := callAndUnmarshalFromURL(&client, server.URL, 1, &calendarData); err == nil {
		t.Fatal("expected an error for non-200 response, got nil")
	}
}

func TestCallAndUnmarshalFromURL_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"events":[`)); err != nil {
			t.Fatalf("failed to write invalid response: %v", err)
		}
	}))
	defer server.Close()

	client := http.Client{Timeout: 2 * time.Second}
	var calendarData APICalendarResponse

	if err := callAndUnmarshalFromURL(&client, server.URL, 1, &calendarData); err == nil {
		t.Fatal("expected json unmarshal error, got nil")
	}
}

func TestGetTime(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		Start string
		End   string
		Err   bool
	}{
		"start_and_end": {
			Start: "2026-03-01T12:00:00-06:00",
			End:   "2026-03-01T13:30:00-06:00",
		},
		"missing_end_uses_start": {
			Start: "2026-03-01T12:00:00-06:00",
			End:   "",
		},
		"invalid_start": {
			Start: "not-a-time",
			End:   "2026-03-01T13:30:00-06:00",
			Err:   true,
		},
		"invalid_end": {
			Start: "2026-03-01T12:00:00-06:00",
			End:   "still-not-a-time",
			Err:   true,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			event := eventWithTimes(testCase.Start, testCase.End)
			start, end, err := getTime(event)

			if testCase.Err {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			expectedStart, _ := time.Parse(time.RFC3339, testCase.Start)
			if !start.Equal(expectedStart) {
				t.Errorf("unexpected start time: got %v, expected %v", start, expectedStart)
			}

			expectedEnd := expectedStart
			if testCase.End != "" {
				expectedEnd, _ = time.Parse(time.RFC3339, testCase.End)
			}
			if !end.Equal(expectedEnd) {
				t.Errorf("unexpected end time: got %v, expected %v", end, expectedEnd)
			}
		})
	}
}

func TestGetEventLocation(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		Event    Event
		Expected string
	}{
		"building_and_room": {
			Event:    Event{Location_name: "SSA", Room_number: "2.406"},
			Expected: "SSA, 2.406",
		},
		"building_only": {
			Event:    Event{Location_name: "SSA", Room_number: ""},
			Expected: "SSA",
		},
		"room_only": {
			Event:    Event{Location_name: "", Room_number: "2.406"},
			Expected: "2.406",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := getEventLocation(testCase.Event); got != testCase.Expected {
				t.Errorf("expected %q, got %q", testCase.Expected, got)
			}
		})
	}
}

func TestGetFiltersAndDepartments(t *testing.T) {
	t.Parallel()

	event := Event{
		Filters: Filters{
			Event_types:           []FilterMap{{Name: "Workshop"}, {Name: "Networking"}},
			Event_target_audience: []FilterMap{{Name: "Students"}},
			Event_topic:           []FilterMap{{Name: "Technology"}, {Name: "Career"}},
		},
		Departments: []FilterMap{{Name: "Engineering"}, {Name: "Career Center"}},
	}

	types, audiences, topics := getFilters(event)
	if len(types) != 2 || types[0] != "Workshop" || types[1] != "Networking" {
		t.Errorf("unexpected event types: %v", types)
	}
	if len(audiences) != 1 || audiences[0] != "Students" {
		t.Errorf("unexpected audiences: %v", audiences)
	}
	if len(topics) != 2 || topics[0] != "Technology" || topics[1] != "Career" {
		t.Errorf("unexpected topics: %v", topics)
	}

	departments := getDepartments(event)
	if len(departments) != 2 || departments[0] != "Engineering" || departments[1] != "Career Center" {
		t.Errorf("unexpected departments: %v", departments)
	}
}

func eventWithTimes(start string, end string) Event {
	return Event{
		Event_instances: []struct {
			Event_instance EventInstance `json:"event_instance"`
		}{
			{
				Event_instance: EventInstance{
					Start: start,
					End:   end,
				},
			},
		},
	}
}
