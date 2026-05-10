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
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"go.mongodb.org/mongo-driver/bson"
)

// Fetch the list of CS courses of 2025 for testing
func FetchTestCourses(outDir string) {
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	var currentCourses []schema.Course

	courseCollection := getCollection(connectDBFunc(), "courses")
	cursor, err := courseCollection.Find(ctx, bson.M{
		"subject_prefix": "CS",
		"catalog_year":   "25",
	})
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

func UploadCourseEmbedding(inDir string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	client := connectDB()

	// Open data file for reading
	file, err := os.Open(fmt.Sprintf("%s/courseEmbeddings.json", inDir))
	if err != nil {
		panic(err)
	}
	defer file.Close()

	UploadData[schema.CourseEmbedding](client, ctx, file, true)
}

type EmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
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

	var semanticInputs []string
	for _, course := range courses {
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
	defer response.Body.Close()

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	var embeddings []schema.CourseEmbedding
	var result EmbeddingResponse
	if err := json.Unmarshal(bytes, &result); err != nil {
		panic(err)
	}
	for i, embedding := range result.Data {
		embeddings = append(embeddings, schema.CourseEmbedding{
			Id:        courses[i].Id,
			Embedding: embedding.Embedding,
		})
	}

	utils.WriteJSON(fmt.Sprintf("%s/courseEmbeddings.json", outDir), embeddings)
}
