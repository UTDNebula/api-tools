package scrapers

import (
	"net/url"
	"testing"
)

func TestParseDelimitedValues(t *testing.T) {
	t.Parallel()

	got := parseDelimitedValues("ecs; bbs, aht ECS")
	if len(got) != 3 {
		t.Fatalf("expected 3 unique values, got %d (%v)", len(got), got)
	}
	if got[0] != "ECS" || got[1] != "BBS" || got[2] != "AHT" {
		t.Fatalf("unexpected parsed values: %v", got)
	}
}

func TestBuildProfileDetailsRequestURL(t *testing.T) {
	t.Parallel()

	raw := buildProfileDetailsRequestURL([]string{"herve.abdi", "nimali.abeykoon"})
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	q := u.Query()

	if q.Get("person") != "herve.abdi;nimali.abeykoon" {
		t.Fatalf("unexpected person query value: %q", q.Get("person"))
	}
	if q.Get("with_data") != "1" {
		t.Fatalf("unexpected with_data query value: %q", q.Get("with_data"))
	}
	if q.Get("data_type") != "information;areas" {
		t.Fatalf("unexpected data_type query value: %q", q.Get("data_type"))
	}
}

func TestBuildProfileSchoolRequestURL(t *testing.T) {
	t.Parallel()

	raw := buildProfileSchoolRequestURL("ecs")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	q := u.Query()

	if q.Get("from_school") != "ECS" {
		t.Fatalf("unexpected from_school query value: %q", q.Get("from_school"))
	}
	if q.Get("with_data") != "1" {
		t.Fatalf("unexpected with_data query value: %q", q.Get("with_data"))
	}
	if q.Get("data_type") != "information;areas" {
		t.Fatalf("unexpected data_type query value: %q", q.Get("data_type"))
	}
}

func TestDedupeProfiles(t *testing.T) {
	t.Parallel()

	items := []profileRawRecord{
		{ID: 1, Slug: "alice.example"},
		{ID: 2, Slug: "ALICE.EXAMPLE"},
		{ID: 3, Slug: ""},
		{ID: 3, Slug: ""},
		{ID: 4, Slug: "bob.example"},
	}

	got := dedupeProfiles(items)
	if len(got) != 3 {
		t.Fatalf("expected 3 unique profiles, got %d", len(got))
	}
}
