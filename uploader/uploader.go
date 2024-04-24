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
			uploadCourses(client, ctx, fptr, replace)
		case "professors.json":
			uploadProfessors(client, ctx, fptr, replace)
		case "sections.json":
			uploadSections(client, ctx, fptr, replace)
		}
	}

}

func uploadCourses(client *mongo.Client, ctx context.Context, fptr *os.File, replace bool) {
	log.Println("Uploading courses.json ...")

	// Decode courses from courses.json
	var courses []schema.Course
	decoder := json.NewDecoder(fptr)
	err := decoder.Decode(&courses)
	if err != nil {
		log.Panic(err)
	}

	if replace {

		// Get collection
		collection := getCollection(client, "courses")

		// Delete all documents from collection
		_, err := collection.DeleteMany(ctx, bson.D{})
		if err != nil {
			log.Panic(err)
		}

		// Convert your courses to []interface{}
		courseDocs := make([]interface{}, len(courses))
		for i := range courses {
			courseDocs[i] = courses[i]
		}

		// Add all documents decoded from courses.json into the collection
		opts := options.InsertMany().SetOrdered(false)
		_, err = collection.InsertMany(ctx, courseDocs, opts)
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

		// Convert your courses to []interface{}
		courseDocs := make([]interface{}, len(courses))
		for i := range courses {
			courseDocs[i] = courses[i]
		}

		// Add all documents decoded from courses.json into the temporary collection
		opts := options.InsertMany().SetOrdered(false)
		_, err = tempCollection.InsertMany(ctx, courseDocs, opts)

		if err != nil {
			log.Panic(err)
		}

		// Create a merge aggregate pipeline
		// Matched documents from the temporary collection will replace matched documents from the Mongo collection
		// Unmatched documents from the temporary collection will be inserted into the Mongo collection
		mergeStage := bson.D{primitive.E{Key: "$merge", Value: bson.D{primitive.E{Key: "into", Value: "courses"}, primitive.E{Key: "on", Value: [3]string{"catalog_year", "course_number", "subject_prefix"}}, primitive.E{Key: "whenMatched", Value: "replace"}, primitive.E{Key: "whenNotMatched", Value: "insert"}}}}

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

	log.Println("Done uploading courses.json!")
}

func uploadProfessors(client *mongo.Client, ctx context.Context, fptr *os.File, replace bool) {
	log.Println("Uploading professors.json ...")

	// Decode courses from professors.json
	var professors []schema.Professor
	decoder := json.NewDecoder(fptr)
	err := decoder.Decode(&professors)
	if err != nil {
		log.Panic(err)
	}

	if replace {

		// Get collection
		collection := getCollection(client, "professors")

		// Delete all documents from collection
		_, err := collection.DeleteMany(ctx, bson.D{})
		if err != nil {
			log.Panic(err)
		}

		// Convert your professors to []interface{}
		professorsDocs := make([]interface{}, len(professors))
		for i := range professors {
			professorsDocs[i] = professors[i]
		}

		// Add all documents decoded from professors.json into the collection
		opts := options.InsertMany().SetOrdered(false)
		_, err = collection.InsertMany(ctx, professorsDocs, opts)
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

		// Convert your professors to []interface{}
		professorsDocs := make([]interface{}, len(professors))
		for i := range professors {
			professorsDocs[i] = professors[i]
		}

		// Add all documents decoded from professors.json into the temporary collection
		opts := options.InsertMany().SetOrdered(false)
		_, err = tempCollection.InsertMany(ctx, professorsDocs, opts)
		if err != nil {
			log.Panic(err)
		}

		// Create a merge aggregate pipeline
		// Matched documents from the temporary collection will replace matched documents from the Mongo collection
		// Unmatched documents from the temporary collection will be inserted into the Mongo collection
		mergeStage := bson.D{primitive.E{Key: "$merge", Value: bson.D{primitive.E{Key: "into", Value: "professors"}, primitive.E{Key: "on", Value: [2]string{"first_name", "last_name"}}, primitive.E{Key: "whenMatched", Value: "replace"}, primitive.E{Key: "whenNotMatched", Value: "insert"}}}}

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

	log.Println("Done uploading professors.json!")
}

func uploadSections(client *mongo.Client, ctx context.Context, fptr *os.File, replace bool) {
	log.Println("Uploading sections.json ...")

	// Decode courses from sections.json
	var sections []schema.Section
	decoder := json.NewDecoder(fptr)
	err := decoder.Decode(&sections)
	if err != nil {
		log.Panic(err)
	}

	if replace {

		// Get collection
		collection := getCollection(client, "sections")

		// Delete all documents from collection
		_, err := collection.DeleteMany(ctx, bson.D{})
		if err != nil {
			log.Panic(err)
		}

		// Convert your sections to []interface{}
		sectionsDocs := make([]interface{}, len(sections))
		for i := range sections {
			sectionsDocs[i] = sections[i]
		}

		// Add all documents decoded from sections.json into the collection
		opts := options.InsertMany().SetOrdered(false)
		_, err = collection.InsertMany(ctx, sectionsDocs, opts)
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

		// Convert your sections to []interface{}
		sectionsDocs := make([]interface{}, len(sections))
		for i := range sections {
			sectionsDocs[i] = sections[i]
		}

		// Add all documents decoded from professors.json into the temporary collection
		opts := options.InsertMany().SetOrdered(false)
		_, err = tempCollection.InsertMany(ctx, sectionsDocs, opts)
		if err != nil {
			log.Panic(err)
		}

		// Create a merge aggregate pipeline
		// Matched documents from the temporary collection will replace matched documents from the Mongo collection
		// Unmatched documents from the temporary collection will be inserted into the Mongo collection
		mergeStage := bson.D{primitive.E{Key: "$merge", Value: bson.D{primitive.E{Key: "into", Value: "sections"}, primitive.E{Key: "on", Value: [3]string{"section_number", "course_reference", "academic_session"}}, primitive.E{Key: "whenMatched", Value: "replace"}, primitive.E{Key: "whenNotMatched", Value: "insert"}}}}

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

	log.Println("Done uploading sections.json!")

	defer fptr.Close()
}
