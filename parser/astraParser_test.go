package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseAstra(t *testing.T) {
	astraTestDir := "testdata"

	// Clean up parser output before starting the test
	os.Remove(filepath.Join(astraTestDir, "astra.json"))

	ParseAstra(astraTestDir, astraTestDir)

	// Check output
	expectedBytes, err := os.ReadFile(filepath.Join(astraTestDir, "expected.json"))
	if err != nil {
		t.Fatalf("Failed to read expected: %v", err)
	}
	outputBytes, err := os.ReadFile(filepath.Join(astraTestDir, "astra.json"))
	if err != nil {
		t.Fatalf("Failed to read parser output: %v", err)
	}

	var output, expected []any

	if err := json.Unmarshal(expectedBytes, &expected); err != nil {
		t.Fatalf("Failed to unmarshal expected")
	}
	if err := json.Unmarshal(outputBytes, &output); err != nil {
		t.Fatalf("Failed to unmarshal output: %v", err)
	}

	if diff := cmp.Diff(expected[0], output[0]); diff != "" {
		t.Errorf("Failed (-expected +got)\n %s", diff)
	}
}
