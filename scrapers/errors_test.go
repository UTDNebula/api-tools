package scrapers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestScrapersReturnOutputDirectoryErrors(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("existing data"), 0600); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(blocked, "output")
	scrapers := map[string]func(string) error{
		"discounts": ScrapeDiscounts, "map": ScrapeMapLocations,
		"degrees": ScrapeDegrees, "academic calendars": ScrapeAcademicCalendars,
		"budgets": ScrapeBudgets, "astra": ScrapeAstra,
		"comet calendar": ScrapeCometCalendar, "mazevo": ScrapeMazevo,
		"profiles": ScrapeProfiles,
	}
	for name, scrape := range scrapers {
		t.Run(name, func(t *testing.T) {
			err := scrape(outDir)
			var pathErr *os.PathError
			if !errors.As(err, &pathErr) {
				t.Fatalf("expected filesystem error, got %v", err)
			}
		})
	}
	content, err := os.ReadFile(blocked)
	if err != nil || string(content) != "existing data" {
		t.Fatalf("existing data changed: %q, %v", content, err)
	}
}

func TestCoursebookReturnsValidationErrors(t *testing.T) {
	for _, args := range []struct{ term, prefix string }{{"26f", "invalid"}, {"invalid", ""}} {
		if err := ScrapeCoursebook(args.term, args.prefix, t.TempDir(), false); err == nil {
			t.Fatal("expected validation error")
		}
	}
}

func TestCoursebookResumeReturnsDirectoryError(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, nil, 0600); err != nil {
		t.Fatal(err)
	}
	scraper := &coursebookScraper{outDir: blocked, term: "26f"}
	if _, err := scraper.lastCompletePrefix(); err == nil {
		t.Fatal("expected output directory error")
	}
}

func TestDownloadPDFReturnsInvalidURLError(t *testing.T) {
	if err := downloadPdf(":invalid", "FY26 Budget", t.TempDir()); err == nil {
		t.Fatal("expected request error")
	}
}
