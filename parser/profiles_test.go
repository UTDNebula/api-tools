package parser

import (
	"testing"
)

func TestBestInformationDataChoosesMostCompleteEntry(t *testing.T) {
	t.Parallel()

	items := []profileInformation{
		{Data: profileInformationData{Email: profileStrPtr(""), Phone: profileStrPtr(""), Location: profileStrPtr("")}},
		{Data: profileInformationData{Email: profileStrPtr("alice@utdallas.edu"), Phone: profileStrPtr("972-000-0000"), URL: profileStrPtr("https://example.com"), Title: profileStrPtr("Professor")}},
		{Data: profileInformationData{Email: profileStrPtr("bob@utdallas.edu")}},
	}

	best := bestInformationData(items)
	if got := trimNullableString(best.Email); got != "alice@utdallas.edu" {
		t.Fatalf("expected most complete information entry, got %q", got)
	}
}

func TestBestProfileURIUsesFallbacks(t *testing.T) {
	t.Parallel()

	row := profileIndexRow{
		APIURL: "https://profiles.utdallas.edu/api/v1?person=alice",
		Information: []profileInformation{
			{Data: profileInformationData{SecondaryURL: profileStrPtr("https://profiles.utdallas.edu/alice")}},
		},
	}

	uri := bestProfileURI(row)
	if uri != "https://profiles.utdallas.edu/alice" {
		t.Fatalf("expected secondary URL fallback, got %q", uri)
	}
}

func TestBestImageURIUsesMediaFallback(t *testing.T) {
	t.Parallel()

	row := profileIndexRow{
		Media: []map[string]any{
			{"id": 11},
			{"image_url": "https://profiles.utdallas.edu/img/alice.jpg"},
		},
	}

	uri := bestImageURI(row)
	if uri != "https://profiles.utdallas.edu/img/alice.jpg" {
		t.Fatalf("expected media image URL fallback, got %q", uri)
	}
}

func TestBuildProfessorFromRowUsesBestLocationAndFallbackURI(t *testing.T) {
	t.Parallel()

	row := profileIndexRow{
		FirstName: "Alice",
		LastName:  "Example",
		Public:    true,
		Information: []profileInformation{
			{Data: profileInformationData{Location: profileStrPtr("Not A Parsable Location"), Email: profileStrPtr("alice@utdallas.edu")}},
			{Data: profileInformationData{Location: profileStrPtr("ECS 3.201"), SecondaryURL: profileStrPtr("https://profiles.utdallas.edu/alice")}},
		},
		Media: []map[string]any{{"url": "https://profiles.utdallas.edu/img/alice2.jpg"}},
	}

	prof := buildProfessorFromRow(row)
	if prof == nil {
		t.Fatal("expected professor to be built")
	}
	if prof.Office.Building != "ECS" || prof.Office.Room != "3.201" {
		t.Fatalf("expected parsed fallback location ECS 3.201, got %+v", prof.Office)
	}
	if prof.Profile_uri != "https://profiles.utdallas.edu/alice" {
		t.Fatalf("expected fallback profile URI, got %q", prof.Profile_uri)
	}
	if prof.Image_uri != "https://profiles.utdallas.edu/img/alice2.jpg" {
		t.Fatalf("expected media fallback image URI, got %q", prof.Image_uri)
	}
}

func profileStrPtr(value string) *string {
	return &value
}
