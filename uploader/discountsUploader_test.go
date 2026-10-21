package uploader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/UTDNebula/nebula-api/rest/schema"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestUploadDiscounts_MissingFile_DoesNotPanic(t *testing.T) {
	// Save original function and restore after test
	originalConnectDB := connectDBFunc
	defer func() { connectDBFunc = originalConnectDB }()

	// Mock DB connection to avoid hitting a real DB
	connectDBFunc = func() *mongo.Client {
		return nil
	}

	tmpDir := t.TempDir()

	// Should not panic when discounts.json is missing
	UploadDiscounts(tmpDir)
}

func TestUploadDiscounts_WithFile_InvokesUploadPath(t *testing.T) {
	// Save original function and restore after test
	originalConnectDB := connectDBFunc
	defer func() { connectDBFunc = originalConnectDB }()

	// Mock DB connection to avoid hitting a real DB
	connectDBFunc = func() *mongo.Client {
		return nil
	}

	tmpDir := t.TempDir()

	// Create a minimal discounts.json so decoding succeeds
	docs := []schema.DiscountProgram{
		{
			Id:       bson.NewObjectID(),
			Category: "Test",
			Business: "Test Business",
			Discount: "10% off",
		},
	}
	b, err := json.Marshal(docs)
	if err != nil {
		t.Fatalf("failed to marshal discounts: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "discounts.json"), b, 0644); err != nil {
		t.Fatalf("failed to write discounts.json: %v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when database operations are attempted")
		}
	}()

	UploadDiscounts(tmpDir)
}
