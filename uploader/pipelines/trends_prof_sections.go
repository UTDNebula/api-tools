package pipelines

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var TrendsProfSectionsPipeline = mongo.Pipeline{
	bson.D{
		{Key: "$lookup",
			Value: bson.D{
				{Key: "from", Value: "sections"},
				{Key: "localField", Value: "sections"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "sections"},
			},
		},
	},
	bson.D{
		{Key: "$project",
			Value: bson.D{
				{Key: "first_name", Value: 1},
				{Key: "last_name", Value: 1},
				{Key: "sections", Value: 1},
			},
		},
	},
}
