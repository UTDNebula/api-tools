package uploader

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TODO: Replace instances and check
type DBSingleton struct {
	client *mongo.Client
}

var dbInstance *DBSingleton
var once sync.Once

func connectDB() *mongo.Client {
	once.Do(func() {

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		opts := options.Client().ApplyURI(getEnvMongoURI())

		client, err := mongo.Connect(ctx, opts)
		if err != nil {
			log.Panic("Unable to create MongoDB client and connect to database")
			os.Exit(1)
		}

		//ping the database
		err = client.Ping(ctx, nil)
		if err != nil {
			log.Panic("Unable to ping database")
			os.Exit(1)
		}

		log.Println("Connected to MongoDB")

		dbInstance = &DBSingleton{
			client: client,
		}
	})

	return dbInstance.client
}

func getCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	collection := client.Database("combinedDB").Collection(collectionName)
	return collection
}

func getEnvMongoURI() string {
	uri, err := utils.GetEnv("MONGODB_URI")
	if err != nil {
		panic(err)
	}
	return uri
}
