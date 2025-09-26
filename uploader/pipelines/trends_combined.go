package pipelines

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var commonLookup = bson.D{
	{Key: "$lookup", Value: bson.D{
		{Key: "from", Value: "sections"},
		{Key: "localField", Value: "sections"},
		{Key: "foreignField", Value: "_id"},
		{Key: "as", Value: "sections"},
	}},
}

func buildPipeline(token string) mongo.Pipeline {
	pipeline := mongo.Pipeline{
		commonLookup,
	}

	switch token {
	case "coursesections":
		pipeline = append(pipeline,
			bson.D{
				{Key: "$project",
					Value: bson.D{
						{Key: "subject_prefix", Value: 1},
						{Key: "course_number", Value: 1},
						{Key: "sections", Value: 1},
					},
				},
			},
			bson.D{
				{Key: "$unwind",
					Value: bson.D{
						{Key: "path", Value: "$sections"},
						{Key: "preserveNullAndEmptyArrays", Value: false},
					},
				},
			},
			bson.D{
				{Key: "$group",
					Value: bson.D{
						{Key: "_id",
							Value: bson.D{
								{Key: "$concat",
									Value: bson.A{
										"$subject_prefix",
										"$course_number",
									},
								},
							},
						},
						{Key: "sections", Value: bson.D{{Key: "$addToSet", Value: "$sections"}}},
					},
				},
			},
		)
	case "profsections":
		pipeline = append(pipeline,
			bson.D{
				{Key: "$project",
					Value: bson.D{
						{Key: "first_name", Value: 1},
						{Key: "last_name", Value: 1},
						{Key: "sections", Value: 1},
					},
				},
			},
		)
	}

	return pipeline
}
