/*
	This file is responsible for handling uploading of parsed event data to MongoDB.
*/

package uploader

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/joho/godotenv"
)

//  Note that this uploader assumes that the collection names match the names of these files, which they should.
//  If the names of these collections ever change, the file names should be updated accordingly.

var eventsFilesToUpload [1]string = [1]string{"astra.json"}

func UploadEvents(inDir string) {

	//Load env vars
	if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
	}

	//Connect to mongo
	client := connectDB()

	// Get 5 minute context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for _, path := range eventsFilesToUpload {

		// Open data file for reading
		fptr, err := os.Open(fmt.Sprintf("%s/"+path, inDir))
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("File not found. Skipping %s", path)
				continue
			}
			log.Panic(err)
		}

		defer fptr.Close()

		switch path {
		case "astra.json":
			UploadData[schema.MultiBuildingEvents[schema.AstraEvent]](client, ctx, fptr, true)
		}
	}
}
