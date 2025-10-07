package uploader

import (
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
)

func TestUpload(t *testing.T) {
	connectDBFunc = func() *mongo.Client {
		return nil // Simple mock, no real connection to db
	}
	defer func() { connectDBFunc = connectDB }() // Point back to original connectDB for any subsequent tests

	Upload("/test", false, true)
}
