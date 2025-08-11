package pipelines

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Pipeline for merging temp collection into existing collection
func MergeStageGenerator(fileName string, matchFilters []string) mongo.Pipeline {
	return mongo.Pipeline{
		bson.D{
			{Key: "$merge", Value: bson.D{
				{Key: "into", Value: fileName},
				{Key: "on", Value: matchFilters},
				{Key: "whenMatched", Value: "replace"},
				{Key: "whenNotMatched", Value: "insert"},
			}},
		},
	}
}
