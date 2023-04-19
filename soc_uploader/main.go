package main

import (
	"context"
	"encoding/csv"
	"fmt"
	_ "github.com/joho/godotenv/autoload"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"strings"
	"time"
)

type Organization struct {
	Title         string   `bson:"title"`
	Categories    []string `bson:"categories"`
	Desc          string   `bson:"description"`
	PresidentName string   `bson:"president_name"`
	ContactEmail  string   `bson:"email"`
	Picture       string   `bson:"picture_url"`
}

// Client instance
var DB = ConnectDB()
var organizationCollection = GetCollection(DB, "organizations")

func main() {
	socdirPath := "socdir.csv"
	file, err := os.Open(socdirPath)
	if err != nil {
		fmt.Printf("Error opening file %s\n", socdirPath)
		os.Exit(1)
	}

	reader := csv.NewReader(file)

	// discard headers
	if _, err = reader.Read(); err != nil {
		fmt.Println("Could not parse contents of file")
		os.Exit(1)
	}

	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Could not parse contents of file")
		os.Exit(1)
	}

	var org Organization
	for _, entry := range records {
		org.Title = entry[0]

		/* parse categories into list of strings */
		cats := entry[1]
		cats = strings.ReplaceAll(cats, "[", "")
		cats = strings.ReplaceAll(cats, "]", "")
		// strange character appears in csv; need to remove it
		cats = strings.ReplaceAll(cats, "\u00a0", "")
		cats = strings.ReplaceAll(cats, `"`, "")
		// split by comma
		catsArray := strings.Split(cats, ",")
		// strip whitespace from ends
		for j, v := range catsArray {
			catsArray[j] = strings.TrimSpace(v)
		}

		org.Categories = catsArray
		org.Desc = entry[2]
		org.PresidentName = entry[3]
		org.ContactEmail = entry[4]
		org.Picture = entry[5]

		InsertIntoDB(&org)
	}

	//for i, entry := range GetAllFromDB()[:10] {
	//	fmt.Printf("Entry %d: %#v\n", i, entry)
	//}
}

func GetAllFromDB() []Organization {
	cursor, err := organizationCollection.Find(context.Background(), bson.D{})
	if err != nil {
		fmt.Println("Error obtaining all orgs from db")
		return nil
	}

	var orgs []Organization

	if err = cursor.All(context.Background(), &orgs); err != nil {
		fmt.Println("Error decoding documents into orgs")
		return nil
	}

	return orgs
}

func InsertIntoDB(org *Organization) {
	_, err := organizationCollection.InsertOne(context.Background(), *org)
	if err != nil {
		fmt.Printf("Error: %v| Could not insert %v\n", err, *org)
	}
}

func GetEnvMongoURI() string {
	//mongoUsername, ok := os.LookupEnv("MONGODB_USERNAME")
	//if !ok {
	//	fmt.Println("Could not find MONGODB_USERNAME")
	//	os.Exit(1)
	//}
	//mongoPassword, ok := os.LookupEnv("MONGODB_PASSWORD")
	//if !ok {
	//	fmt.Println("Could not find MONGODB_PASSWORD")
	//	os.Exit(1)
	//}
	//mongoPassword = url.QueryEscape(mongoPassword)
	//mongoCluster, ok := os.LookupEnv("MONGODB_CLUSTER")
	//if !ok {
	//	fmt.Println("Could not find MONGODB_CLUSTER")
	//	os.Exit(1)
	//}
	//mongoUri := fmt.Sprintf("mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority",
	//	mongoUsername, mongoPassword, mongoCluster)
	uri, exist := os.LookupEnv("MONGODB_URI")
	if !exist {
		log.Fatalf("Error loading 'MONGODB_URI' from the .env file")
	}

	return uri
	//return mongoUri
}
func ConnectDB() *mongo.Client {
	client, err := mongo.NewClient(options.Client().ApplyURI(GetEnvMongoURI()))
	if err != nil {
		log.Fatalf("Unable to create MongoDB client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	//ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}
	fmt.Println("Connected to MongoDB")
	return client
}

// getting database collections
func GetCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	collection := client.Database("combinedDB").Collection(collectionName)
	return collection
}
