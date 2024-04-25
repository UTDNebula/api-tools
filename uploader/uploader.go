/*
	This file is responsible for handling uploading of parsed data to MongoDB.
*/

package uploader

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/joho/godotenv"
)

//  It's important to note that all of the files must be updated/uploaded TOGETHER!
//  This is because the parser links all of the data together with ObjectID references, and
//  these references will change and cause things to break if files are updated/uploaded individually!

//  Also note that this uploader assumes that the collection names match the names of these files, which they should.
//  If the names of these collections ever change, the file names should be updated accordingly.

var filesToUpload [3]string = [3]string{"courses.json", "professors.json", "sections.json"}

func Upload(inDir string, replace bool) {

	//Load env vars
	if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
	}

	//Connect to mongo
	client := connectDB()

	// Get 5 minute context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for _, path := range filesToUpload {

		// Open data file for reading
		fptr, err := os.Open(fmt.Sprintf("%s/"+path, inDir))
		if err != nil {
			log.Panic(err)
		}

		defer fptr.Close()

		switch path {
		case "courses.json":
			UploadData[schema.Course](client, ctx, fptr, replace)
		case "professors.json":
			UploadData[schema.Professor](client, ctx, fptr, replace)
		case "sections.json":
			UploadData[schema.Section](client, ctx, fptr, replace)
		}
	}

}

// Generic upload function to upload parsed JSON data to the Mongo database
// Make sure that the name of the file being parsed matches with the name of the collection you are uploading to!
// For example, your file should be named courses.json if you want to upload courses
// As of right now, courses, professors, and sections are available to upload.
func UploadData[T any](client *mongo.Client, ctx context.Context, fptr *os.File, replace bool) {
	fileName := fptr.Name()[strings.LastIndex(fptr.Name(), "/")+1 : len(fptr.Name())-5]
	log.Println("Uploading " + fileName + ".json ...")

	// Decode documents from file
	var docs []T
	decoder := json.NewDecoder(fptr)
	err := decoder.Decode(&docs)
	if err != nil {
		log.Panic(err)
	}

	if replace {

		// Get collection
		collection := getCollection(client, fileName)

		// Delete all documents from collection
		_, err := collection.DeleteMany(ctx, bson.D{})
		if err != nil {
			log.Panic(err)
		}

		// Convert your documents to []interface{}
		docsInterface := make([]interface{}, len(docs))
		for i := range docs {
			docsInterface[i] = docs[i]
		}

		// Add all documents decoded from the file into the collection
		opts := options.InsertMany().SetOrdered(false)
		_, err = collection.InsertMany(ctx, docsInterface, opts)
		if err != nil {
			log.Panic(err)
		}

	} else {
		// If a temp collection already exists, drop it
		tempCollection := getCollection(client, "temp")
		err = tempCollection.Drop(ctx)
		if err != nil {
			log.Panic(err)
		}

		// Create a temporary collection
		err := client.Database("combinedDB").CreateCollection(ctx, "temp")
		if err != nil {
			log.Panic(err)
		}

		// Get the temporary collection
		tempCollection = getCollection(client, "temp")

		// Convert your documents to []interface{}
		docsInterface := make([]interface{}, len(docs))
		for i := range docs {
			docsInterface[i] = docs[i]
		}

		// Add all documents decoded from the file into the temporary collection
		opts := options.InsertMany().SetOrdered(false)
		_, err = tempCollection.InsertMany(ctx, docsInterface, opts)
		if err != nil {
			log.Panic(err)
		}

		// Create a merge aggregate pipeline
		// Matched documents from the temporary collection will replace matched documents from the Mongo collection
		// Unmatched documents from the temporary collection will be inserted into the Mongo collection
		var matchFilters []string
		switch fileName {
		case "courses":
			matchFilters = []string{"catalog_year", "course_number", "subject_prefix"}
		case "professors":
			matchFilters = []string{"first_name", "last_name"}
		case "sections":
			matchFilters = []string{"section_number", "course_reference", "academic_session"}
		default:
			log.Panic("Unrecognizable filename: " + fileName)
		}

		// The documents will be added/merged into the collection with the same name as the file
		// The filters for the merge aggregate pipeline are based on the file name
		mergeStage := bson.D{primitive.E{Key: "$merge", Value: bson.D{primitive.E{Key: "into", Value: fileName}, primitive.E{Key: "on", Value: matchFilters}, primitive.E{Key: "whenMatched", Value: "replace"}, primitive.E{Key: "whenNotMatched", Value: "insert"}}}}

		// Execute aggregate pipeline
		_, err = tempCollection.Aggregate(ctx, mongo.Pipeline{mergeStage})
		if err != nil {
			log.Panic(err)
		}

		// Drop the temporary collection
		err = tempCollection.Drop(ctx)
		if err != nil {
			log.Panic(err)
		}
	}

	log.Println("Done uploading " + fileName + ".json!")

	defer fptr.Close()
}
