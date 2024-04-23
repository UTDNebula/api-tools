/*
	This file is responsible for providing various useful database functions.
*/

package uploader

import (
	//"go.mongodb.org/mongo-driver/bson"
	//"go.mongodb.org/mongo-driver/bson/primitive"
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func connectDB() *mongo.Client {
	client, err := mongo.NewClient(options.Client().ApplyURI(getEnvMongoURI()))
	if err != nil {
		log.Panic("Unable to create MongoDB client")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Panic("Unable to connect to database")
		os.Exit(1)
	}

	//ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Panic("Unable to ping database")
		os.Exit(1)
	}

	log.Println("Connected to MongoDB")

	return client
}

func getCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	collection := client.Database("combinedDB").Collection(collectionName)
	return collection
}

func getEnvMongoURI() string {
	uri, exist := os.LookupEnv("MONGODB_URI")
	if !exist {
		log.Panic("Error loading 'MONGODB_URI' from the .env file")
		os.Exit(1)
	}
	return uri
}
