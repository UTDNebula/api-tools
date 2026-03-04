package scrapers

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExtractSectionIDs_FromFixture(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile(filepath.Join("testdata", "coursebook", "search-results-sample.html"))
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	ids := extractSectionIDs("cp_acct", string(content))
	expected := []string{
		"acct2301.001.25S",
		"acct2301.002.25S",
		"acct6v01.0W1.25S",
	}

	if !reflect.DeepEqual(expected, ids) {
		t.Errorf("unexpected ids. expected %v, got %v", expected, ids)
	}
}

func TestExtractSectionIDs_NoMatches(t *testing.T) {
	t.Parallel()

	content := `<html><body><div>View details for section cs1337.001.25S</div></body></html>`
	ids := extractSectionIDs("cp_acct", content)

	if len(ids) != 0 {
		t.Errorf("expected no ids, got %v", ids)
	}
}

func TestGetMissingIdsForPrefix_NoDirectory(t *testing.T) {
	t.Parallel()

	ids := []string{"acct2301.001.25S", "acct2301.002.25S"}
	scraper := &coursebookScraper{
		term:           "25S",
		outDir:         t.TempDir(),
		prefixIdsCache: map[string][]string{"cp_acct": ids},
	}

	missing, err := scraper.getMissingIdsForPrefix("cp_acct")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !reflect.DeepEqual(ids, missing) {
		t.Errorf("expected all ids to be missing. expected %v, got %v", ids, missing)
	}
}

func TestGetMissingIdsForPrefix_FiltersExistingFiles(t *testing.T) {
	t.Parallel()

	ids := []string{"acct2301.001.25S", "acct2301.002.25S", "acct2301.003.25S"}
	outDir := t.TempDir()
	prefixDir := filepath.Join(outDir, "25S", "cp_acct")

	if err := os.MkdirAll(prefixDir, 0755); err != nil {
		t.Fatalf("failed to create prefix directory: %v", err)
	}

	existing := "acct2301.002.25S"
	if err := os.WriteFile(filepath.Join(prefixDir, existing+".html"), []byte("cached"), 0644); err != nil {
		t.Fatalf("failed to seed existing section file: %v", err)
	}

	scraper := &coursebookScraper{
		term:           "25S",
		outDir:         outDir,
		prefixIdsCache: map[string][]string{"cp_acct": ids},
	}

	missing, err := scraper.getMissingIdsForPrefix("cp_acct")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []string{"acct2301.001.25S", "acct2301.003.25S"}
	if !reflect.DeepEqual(expected, missing) {
		t.Errorf("unexpected missing ids. expected %v, got %v", expected, missing)
	}
}
