/*
	This file is the POC for embedding the course data to perform semantic searching
*/

package uploader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/controllers"
	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const COURSE_VECTOR_INDEX = "course_embedding_index"

// Fetch the list of CS courses of 2025 for testing
func FetchTestCourses(outDir string) {
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	var currentCourses []schema.Course

	courseCollection := getCollection(connectDBFunc(), "courses")
	cursor, err := courseCollection.Find(ctx, bson.M{"catalog_year": "25"})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &currentCourses); err != nil {
		log.Fatal(err)
	}

	// Make outDir if it doesn't already exist
	err = os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	utils.WriteJSON(fmt.Sprintf("%s/courseEmbeddingInputs.json", outDir), currentCourses)
}

// UploadCourseEmbedding uploads the course embedding data into the MongoDB collection
func UploadCourseEmbedding(inDir string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client := connectDB()

	// Check if the vector index exists
	embeddingCollection := getCollection(client, "courseEmbeddings")
	opts := options.SearchIndexes().SetName(COURSE_VECTOR_INDEX)
	var vectorIndexes []bson.M
	cursor, err := embeddingCollection.SearchIndexes().List(ctx, opts)
	if err != nil {
		panic(err)
	}
	defer cursor.Close(ctx)
	if err = cursor.All(ctx, &vectorIndexes); err != nil {
		panic(err)
	}
	if len(vectorIndexes) == 0 {
		log.Println("Vector index doesn't exist, creating one...")

		// Create vector index if it doesn't exist yet
		definition := bson.D{
			{Key: "mappings", Value: bson.D{
				{Key: "dynamic", Value: false},
				{Key: "fields", Value: bson.D{
					{Key: "embedding", Value: bson.D{
						{Key: "type", Value: "vector"},
						{Key: "numDimensions", Value: 1024},
						{Key: "similarity", Value: "cosine"},
					}},
				}},
			}},
		}
		_, err = embeddingCollection.SearchIndexes().CreateOne(ctx, mongo.SearchIndexModel{
			Definition: definition,
			Options:    opts,
		})
		if err != nil {
			panic(err)
		}

		// TODO: Wait until it's queryable

		log.Println("Created vector index!")
	}

	// Open data file for reading
	file, err := os.Open(fmt.Sprintf("%s/courseEmbeddings.json", inDir))
	if err != nil {
		panic(err)
	}
	defer file.Close()

	UploadData[schema.CourseEmbedding](client, ctx, file, true)
}

// Embed the list of courses
func CourseEmbedder(inDir string, outDir string) {
	embeddingURL := "https://ai.mongodb.com/v1/embeddings"
	embeddingKey, err := utils.GetEnv("EMBEDDING_KEY")
	if err != nil {
		panic(err)
	}
	var courses []schema.Course

	// Get the list of inputs
	file, err := os.Open(fmt.Sprintf("%s/courseEmbeddingInputs.json", inDir))
	if err != nil {
		panic(err)
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&courses)
	if err != nil {
		panic(err)
	}

	log.Printf("Embedding %d course data...\n", len(courses))

	var embeddings []schema.CourseEmbedding

	const batchSize = 100
	batchStart := 0
	for courseBatch := range slices.Chunk(courses, batchSize) {
		var semanticInputs []string
		for _, course := range courseBatch {
			// TODO: Need to put syllabus data into this
			semanticInputs = append(semanticInputs, fmt.Sprintf(
				"Subject: %s. Course: %s. Description: %s. Level: %s.\n",
				course.Subject_prefix,
				course.Title,
				course.Description,
				course.Class_level,
			))
		}
		body, _ := json.Marshal(map[string]any{
			"input": semanticInputs,
			"model": "voyage-4-large",
		})

		request, _ := http.NewRequest("POST", embeddingURL, bytes.NewBuffer(body))
		request.Header.Set("Authorization", "Bearer "+embeddingKey)
		request.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			panic(err)
		}

		bytes, err := io.ReadAll(response.Body)
		if err != nil {
			panic(err)
		}
		response.Body.Close()

		var result controllers.EmbeddingResponse
		if err := json.Unmarshal(bytes, &result); err != nil {
			panic(err)
		}
		for i, embedding := range result.Data {
			embeddings = append(embeddings, schema.CourseEmbedding{
				Id:        courses[i+batchStart].Id,
				Embedding: embedding.Embedding,
			})
		}

		batchStart += batchSize
	}

	utils.WriteJSON(fmt.Sprintf("%s/courseEmbeddings.json", outDir), embeddings)
	log.Println("Embeded course data!")
}
