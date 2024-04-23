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

	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/joho/godotenv"
)

//  It's important to note that all of the files must be updated/uploaded TOGETHER!
//  This is because the parser links all of the data together with ObjectID references, and
//  these references will change and cause things to break if files are updated/uploaded individually!

//  Also note that this uploader assumes that the collection names match the names of these files, which they should.
//  If the names of these collections ever change, the file names should be updated accordingly.

var filesToUpload []string = []string{"courses.json", "professors.json", "sections.json"}

func Upload(inDir string, replace bool) {

	//Load env vars
	if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
	}

	//Connect to mongo
	client := connectDB()

	// Get context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, path := range filesToUpload {
		// Open data file for reading
		fptr, err := os.Open(fmt.Sprintf("%s/"+path, inDir))
		if err != nil {
			log.Panic(err)
		}

		switch path {
		case "courses.json":
			fmt.Println("Uploading courses.json ...")

			// Decode courses from courses.json
			var courses []schema.Course
			decoder := json.NewDecoder(fptr)
			err = decoder.Decode(&courses)
			if err != nil {
				log.Panic(err)
			}

			if replace {
				var empty interface{}

				// Get collection
				collection := getCollection(client, "courses")

				// Delete all documents from collection
				_, err := collection.DeleteMany(ctx, empty)
				if err != nil {
					log.Panic(err)
				}

				// Add all documents decoded from courses.json into Mongo collection
				for _, course := range courses {
					_, err := collection.InsertOne(ctx, course)
					if err != nil {
						log.Panic(err)
					}
				}
			} else {
				// Create a temporary collection
				err := client.Database("combinedDB").CreateCollection(ctx, "temp")
				if err != nil {
					log.Panic(err)
				}

				// Get the temporary collection
				tempCollection := getCollection(client, "temp")

				// Add all documents decoded from courses.json into the temporary collection
				for _, course := range courses {
					_, err := tempCollection.InsertOne(ctx, course)
					if err != nil {
						log.Panic(err)
					}
				}

				// Create a merge aggregate pipeline
				// Matched documents from the temporary collection will replace matched documents from the Mongo collection
				// Unmatched documents from the temporary collection will be inserted into the Mongo collection
				mergeStage := bson.D{primitive.E{Key: "$merge", Value: bson.D{primitive.E{Key: "into", Value: "courses"}, primitive.E{Key: "on", Value: []string{"catalog_year", "course_number", "subject_prefix"}}, primitive.E{Key: "whenMatched", Value: "replace"}, primitive.E{Key: "whenNotMatched", Value: "insert"}}}}

				// Execute aggregate pipeline
				_, err = tempCollection.Aggregate(ctx, mergeStage)
				if err != nil {
					log.Panic(err)
				}

				// Drop the temporary collection
				err = tempCollection.Drop(ctx)
				if err != nil {
					log.Panic(err)
				}
			}

			fmt.Println("Done uploading courses.json!")

		case "professors.json":
			fmt.Println("Uploading professors.json ...")

			// Decode courses from professors.json
			var professors []schema.Professor
			decoder := json.NewDecoder(fptr)
			err = decoder.Decode(&professors)
			if err != nil {
				log.Panic(err)
			}

			if replace {
				var empty interface{}

				// Get collection
				collection := getCollection(client, "professors")

				// Delete all documents from collection
				_, err := collection.DeleteMany(ctx, empty)
				if err != nil {
					log.Panic(err)
				}

				// Add all documents decoded from professors.json into Mongo collection
				for _, professor := range professors {
					_, err := collection.InsertOne(ctx, professor)
					if err != nil {
						log.Panic(err)
					}
				}
			} else {
				// Create a temporary collection
				err := client.Database("combinedDB").CreateCollection(ctx, "temp")
				if err != nil {
					log.Panic(err)
				}

				// Get the temporary collection
				tempCollection := getCollection(client, "temp")

				// Add all documents decoded from professors.json into the temporary collection
				for _, professor := range professors {
					_, err = tempCollection.InsertOne(ctx, professor)
					if err != nil {
						log.Panic(err)
					}
				}

				// Create a merge aggregate pipeline
				// Matched documents from the temporary collection will replace matched documents from the Mongo collection
				// Unmatched documents from the temporary collection will be inserted into the Mongo collection
				mergeStage := bson.D{primitive.E{Key: "$merge", Value: bson.D{primitive.E{Key: "into", Value: "professors"}, primitive.E{Key: "on", Value: []string{"first_name", "last_name"}}, primitive.E{Key: "whenMatched", Value: "replace"}, primitive.E{Key: "whenNotMatched", Value: "insert"}}}}

				// Execute aggregate pipeline
				_, err = tempCollection.Aggregate(ctx, mergeStage)
				if err != nil {
					log.Panic(err)
				}

				// Drop the temporary collection
				err = tempCollection.Drop(ctx)
				if err != nil {
					log.Panic(err)
				}
			}

			fmt.Println("Done uploading professors.json!")

		case "sections.json":
			fmt.Println("Uploading sections.json ...")

			// Decode courses from sections.json
			var sections []schema.Section
			decoder := json.NewDecoder(fptr)
			err = decoder.Decode(&sections)
			if err != nil {
				log.Panic(err)
			}

			if replace {
				var empty interface{}

				// Get collection
				collection := getCollection(client, "sections")

				// Delete all documents from collection
				_, err := collection.DeleteMany(ctx, empty)
				if err != nil {
					log.Panic(err)
				}

				// Add all documents decoded from sections.json into Mongo collection
				for _, section := range sections {
					_, err := collection.InsertOne(ctx, section)
					if err != nil {
						log.Panic(err)
					}
				}
			} else {
				// Create a temporary collection
				err := client.Database("combinedDB").CreateCollection(ctx, "temp")
				if err != nil {
					log.Panic(err)
				}

				// Get the temporary collection
				tempCollection := getCollection(client, "temp")

				// Add all documents decoded from sections.json into the temporary collection
				for _, section := range sections {
					_, err := tempCollection.InsertOne(ctx, section)
					if err != nil {
						log.Panic(err)
					}
				}

				// Create a merge aggregate pipeline
				// Matched documents from the temporary collection will replace matched documents from the Mongo collection
				// Unmatched documents from the temporary collection will be inserted into the Mongo collection
				mergeStage := bson.D{primitive.E{Key: "$merge", Value: bson.D{primitive.E{Key: "into", Value: "sections"}, primitive.E{Key: "on", Value: []string{"section_number", "course_reference", "academic_session"}}, primitive.E{Key: "whenMatched", Value: "replace"}, primitive.E{Key: "whenNotMatched", Value: "insert"}}}}

				// Execute aggregate pipeline
				_, err = tempCollection.Aggregate(ctx, mergeStage)
				if err != nil {
					log.Panic(err)
				}

				// Drop the temporary collection
				err = tempCollection.Drop(ctx)
				if err != nil {
					log.Panic(err)
				}
			}

			fmt.Println("Done uploading sections.json!")
		}

		defer fptr.Close()
	}

}
