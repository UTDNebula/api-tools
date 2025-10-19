package pipelines

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var TrendsCourseProfSectionsPipeline = mongo.Pipeline{
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
		{Key: "$lookup",
			Value: bson.D{
				{Key: "from", Value: "professors"},
				{Key: "localField", Value: "sections.professors"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "professors"},
			},
		},
	},
	bson.D{
		{Key: "$unwind",
			Value: bson.D{
				{Key: "path", Value: "$professors"},
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
								" ",
								"$professors.first_name",
								" ",
								"$professors.last_name",
							},
						},
					},
				},
				{Key: "sections", Value: bson.D{{Key: "$addToSet", Value: "$sections"}}},
			},
		},
	},
}
