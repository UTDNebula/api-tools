package scrapers

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func docFromHTML(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to parse test HTML: %v", err)
	}
	return doc
}

func TestParseFacilitiesCentersAndMerge(t *testing.T) {
	html := `
		<html><body>
			<div class="entry-content">
				<h2 class="wp-block-heading">University-Wide Centers and Institutes</h2>
				<ul class="wp-block-list">
					<li><a href="/uw">UW Center</a></li>
				</ul>

				<h2 class="wp-block-heading">Research Centers and Institutes</h2>
				<h3 class="wp-block-heading">School A</h3>
				<ul class="wp-block-list">
					<li><a href="https://example.com/c2">Center Two</a></li>
				</ul>
				<h3 class="wp-block-heading">School B</h3>
				<ul class="wp-block-list">
					<li><a href="https://example.com/c2">Center Two</a></li>
				</ul>

				<h3 class="wp-block-heading">Related Pages</h3>
				<ul class="wp-block-list">
					<li><a href="https://example.com/related">Not a listing</a></li>
				</ul>
			</div>
		</body></html>
	`

	listings := parseFacilitiesCenters(docFromHTML(t, html))
	listings = mergeResearchListings(listings)

	if len(listings) != 2 {
		t.Fatalf("expected 2 listings, got %d", len(listings))
	}

	byName := make(map[string]ResearchListing, len(listings))
	for _, l := range listings {
		byName[l.Name] = l
	}

	uw, ok := byName["UW Center"]
	if !ok {
		t.Fatalf("missing UW Center listing")
	}
	if uw.Link != "https://www.utdallas.edu/uw" {
		t.Fatalf("unexpected UW Center link: %q", uw.Link)
	}
	if uw.School != "University-Wide" {
		t.Fatalf("unexpected UW Center school: %q", uw.School)
	}
	if uw.Source != "facilities-centers" {
		t.Fatalf("unexpected UW Center source: %q", uw.Source)
	}
	if uw.Id == primitive.NilObjectID {
		t.Fatalf("expected UW Center _id to be set")
	}

	c2, ok := byName["Center Two"]
	if !ok {
		t.Fatalf("missing Center Two listing")
	}
	if c2.School != "School A | School B" {
		t.Fatalf("unexpected Center Two school: %q", c2.School)
	}

	if _, ok := byName["Not a listing"]; ok {
		t.Fatalf("expected Related Pages listing to be skipped")
	}
}

func TestParseLabs(t *testing.T) {
	html := `
		<html><body>
			<div class="entry-content">
				<h2 class="wp-block-heading">School of Testing</h2>
				<ul class="wp-block-list">
					<li><a href="lab1/">Lab One</a> (Prof A and Prof B)</li>
					<li><a href="lab2/">Lab Two</a> (Prof C &amp; Prof D)</li>
					<li><a href="lab3/">Lab Three</a></li>
				</ul>
			</div>
		</body></html>
	`

	listings := parseLabs(docFromHTML(t, html))
	if len(listings) != 3 {
		t.Fatalf("expected 3 listings, got %d", len(listings))
	}

	byName := make(map[string]ResearchListing, len(listings))
	for _, l := range listings {
		byName[l.Name] = l
	}

	one := byName["Lab One"]
	if one.School != "School of Testing" {
		t.Fatalf("unexpected Lab One school: %q", one.School)
	}
	if one.Link != "https://labs.utdallas.edu/lab1" {
		t.Fatalf("unexpected Lab One link: %q", one.Link)
	}
	if one.Source != "labs" {
		t.Fatalf("unexpected Lab One source: %q", one.Source)
	}
	if len(one.Professors) != 2 || one.Professors[0] != "Prof A" || one.Professors[1] != "Prof B" {
		t.Fatalf("unexpected Lab One professors: %#v", one.Professors)
	}

	two := byName["Lab Two"]
	if len(two.Professors) != 2 || two.Professors[0] != "Prof C" || two.Professors[1] != "Prof D" {
		t.Fatalf("unexpected Lab Two professors: %#v", two.Professors)
	}

	three := byName["Lab Three"]
	if len(three.Professors) != 0 {
		t.Fatalf("expected Lab Three professors to be empty, got %#v", three.Professors)
	}
}

func TestMergeResearchListings_MergesSourcesSchoolsAndCanonicalURLs(t *testing.T) {
	listings := []ResearchListing{
		{
			Id:     primitive.NewObjectID(),
			Name:   "Example Lab",
			Link:   "https://example.com/lab/",
			School: "School A",
			Source: "facilities-centers",
		},
		{
			Id:         primitive.NewObjectID(),
			Name:       "Example Lab",
			Link:       "https://example.com/lab",
			School:     "School B",
			Professors: []string{"Prof A"},
			Source:     "labs",
		},
	}

	merged := mergeResearchListings(listings)
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged listing, got %d", len(merged))
	}

	got := merged[0]
	if got.Link != "https://example.com/lab" {
		t.Fatalf("unexpected merged link: %q", got.Link)
	}
	if got.School != "School A | School B" {
		t.Fatalf("unexpected merged school: %q", got.School)
	}
	if got.Source != "facilities-centers | labs" {
		t.Fatalf("unexpected merged source: %q", got.Source)
	}
	if len(got.Professors) != 1 || got.Professors[0] != "Prof A" {
		t.Fatalf("unexpected merged professors: %#v", got.Professors)
	}
}
