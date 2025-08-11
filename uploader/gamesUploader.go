/*
	This file is responsible for handling uploading of game data to MongoDB.
*/

package uploader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
)

//  Note that this uploader assumes that the collection names match the names of these files, which they should.
//  If the names of these collections ever change, the file names should be updated accordingly.

var lettersFile string = "letters.json"

func UploadLetters(inDir string, replace bool) {

	//Load env vars
	if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
	}

	//Connect to mongo
	client := connectDB()

	// Get 5 minute context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Open data file for reading
	fptr, err := os.Open(fmt.Sprintf("%s/"+lettersFile, inDir))
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("File not found. Skipping %s", lettersFile)
			return
		}
		log.Panic(err)
	}
	defer fptr.Close()

	if replace {
		UploadData[schema.Letters](client, ctx, fptr, false)
	} else {
		// Get date to upload
		var docs []schema.Letters
		decoder := json.NewDecoder(fptr)
		err := decoder.Decode(&docs)
		if err != nil {
			log.Panic(err)
		}
		if len(docs) != 1 {
			log.Println("0 or 2+ entries found in JSON, skipping upload.")
			return
		}
		today := docs[0].Date

		// Check if date already exists
		fileName := fptr.Name()[strings.LastIndex(fptr.Name(), "/")+1 : len(fptr.Name())-5]
		collection := getCollection(client, fileName)
		filter := bson.M{"date": today}
		count, err := collection.CountDocuments(ctx, filter)
		if err != nil {
			log.Panicf("Error checking for existing puzzle: %v", err)
		}
		if count > 0 {
			log.Printf("Puzzle for %s already exists. Skipping upload.", today)
			return
		}

		fptr.Seek(0, io.SeekStart)
		UploadData[schema.Letters](client, ctx, fptr, false)
	}
}
