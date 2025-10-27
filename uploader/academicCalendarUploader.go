/*
	This file is responsible for handling uploading of parsed academic calendar data to MongoDB.
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

var academicCalendarFile = "academicCalendars.json"

func UploadAcademicCalendars(inDir string) {

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
	fptr, err := os.Open(fmt.Sprintf("%s/"+academicCalendarFile, inDir))
	if err != nil {
		if os.IsNotExist(err) {
			log.Panicf("File not found. Skipping %s", academicCalendarFile)
		}
		log.Panic(err)
	}

	defer fptr.Close()

	UploadData[schema.AcademicCalendar](client, ctx, fptr, true)
}
