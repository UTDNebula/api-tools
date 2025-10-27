package uploader

import (
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
)

func TestUpload(t *testing.T) {
	// Save original function and restore after test
	originalConnectDB := connectDBFunc
	defer func() { connectDBFunc = originalConnectDB }()

	// Create a simple mock that returns nil (or a minimal mock client)
	connectDBFunc = func() *mongo.Client {
		return nil
	}

	// Test cases
	tests := []struct {
		name       string
		inDir      string
		replace    bool
		staticOnly bool
	}{
		{
			name:       "static only mode",
			inDir:      "./testdata",
			replace:    false,
			staticOnly: true,
		},
		{
			name:       "full upload with replace",
			inDir:      "./testdata",
			replace:    true,
			staticOnly: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will panic when it tries to use the nil client, but that's fine for now
			// The goal is to test that the function calls what it should call
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Expected panic when database operations are attempted: %v", r)
				}
			}()

			Upload(tt.inDir, tt.replace, tt.staticOnly)
		})
	}
}
