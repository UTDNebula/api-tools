package pipelines

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Pipeline for aggregating sections->events
var EventsPipeline = mongo.Pipeline{
	//separate each meeting
	{{Key: "$unwind", Value: "$meetings"}},
	//remove bad data
	{{Key: "$match", Value: bson.D{
		{Key: "meetings.location.building", Value: bson.D{{Key: "$ne", Value: "No"}}},
	}}},
	//remove time from meeting start date
	{{Key: "$addFields", Value: bson.D{
		{Key: "meetings.start_date", Value: bson.D{
			{Key: "$dateTrunc", Value: bson.D{
				{Key: "date", Value: "$meetings.start_date"},
				{Key: "unit", Value: "day"},
			}},
		}},
	}}},
	//add a field thats a list of all days the class occurs
	//by generating all days in semester and removing those that don't match MW for example
	{{Key: "$addFields", Value: bson.D{
		{Key: "meeting_dates", Value: bson.D{
			{Key: "$filter", Value: bson.D{
				{Key: "input", Value: bson.D{
					{Key: "$map", Value: bson.D{
						{Key: "input", Value: bson.D{
							{Key: "$range", Value: bson.A{
								0,
								bson.D{{Key: "$dateDiff", Value: bson.D{
									{Key: "startDate", Value: bson.D{{Key: "$toDate", Value: "$meetings.start_date"}}},
									{Key: "endDate", Value: bson.D{{Key: "$toDate", Value: "$meetings.end_date"}}},
									{Key: "unit", Value: "day"},
								}}},
								1,
							}},
						}},
						{Key: "as", Value: "offset"},
						{Key: "in", Value: bson.D{
							{Key: "$dateAdd", Value: bson.D{
								{Key: "startDate", Value: bson.D{{Key: "$toDate", Value: "$meetings.start_date"}}},
								{Key: "unit", Value: "day"},
								{Key: "amount", Value: "$$offset"},
							}},
						}},
					}},
				}},
				{Key: "as", Value: "date"},
				{Key: "cond", Value: bson.D{
					{Key: "$in", Value: bson.A{
						bson.D{{Key: "$dayOfWeek", Value: "$$date"}},
						bson.D{{Key: "$map", Value: bson.D{
							{Key: "input", Value: "$meetings.meeting_days"},
							{Key: "as", Value: "day"},
							{Key: "in", Value: bson.D{
								{Key: "$switch", Value: bson.D{
									{Key: "branches", Value: bson.A{
										bson.D{{Key: "case", Value: bson.D{{Key: "$eq", Value: bson.A{"$$day", "Sunday"}}}}, {Key: "then", Value: 1}},
										bson.D{{Key: "case", Value: bson.D{{Key: "$eq", Value: bson.A{"$$day", "Monday"}}}}, {Key: "then", Value: 2}},
										bson.D{{Key: "case", Value: bson.D{{Key: "$eq", Value: bson.A{"$$day", "Tuesday"}}}}, {Key: "then", Value: 3}},
										bson.D{{Key: "case", Value: bson.D{{Key: "$eq", Value: bson.A{"$$day", "Wednesday"}}}}, {Key: "then", Value: 4}},
										bson.D{{Key: "case", Value: bson.D{{Key: "$eq", Value: bson.A{"$$day", "Thursday"}}}}, {Key: "then", Value: 5}},
										bson.D{{Key: "case", Value: bson.D{{Key: "$eq", Value: bson.A{"$$day", "Friday"}}}}, {Key: "then", Value: 6}},
										bson.D{{Key: "case", Value: bson.D{{Key: "$eq", Value: bson.A{"$$day", "Saturday"}}}}, {Key: "then", Value: 7}},
									}},
									{Key: "default", Value: nil},
								}},
							}},
						}}},
					}},
				}},
			}},
		}},
	}}},
	//separate for each meeting data
	{{Key: "$unwind", Value: "$meeting_dates"}},
	//remove data before today
	{{Key: "$match", Value: bson.D{
		{Key: "$expr", Value: bson.D{
			{Key: "$gte", Value: bson.A{
				"$meeting_dates",
				bson.D{{Key: "$dateSubtract", Value: bson.D{
					{Key: "startDate", Value: bson.D{{Key: "$dateTrunc", Value: bson.D{
						{Key: "date", Value: "$$NOW"},
						{Key: "unit", Value: "day"},
					}}}},
					{Key: "unit", Value: "day"},
					{Key: "amount", Value: 7},
				}}},
			}},
		}},
	}}},
	//group into building > room hierarchy format
	{{Key: "$group", Value: bson.D{
		{Key: "_id", Value: bson.D{
			{Key: "date", Value: "$meeting_dates"},
			{Key: "building", Value: "$meetings.location.building"},
			{Key: "room", Value: "$meetings.location.room"},
		}},
		{Key: "events", Value: bson.D{
			{Key: "$push", Value: bson.D{
				{Key: "section", Value: "$_id"},
				{Key: "start_time", Value: "$meetings.start_time"},
				{Key: "end_time", Value: "$meetings.end_time"},
			}},
		}},
	}}},
	{{Key: "$group", Value: bson.D{
		{Key: "_id", Value: bson.D{
			{Key: "date", Value: "$_id.date"},
			{Key: "building", Value: "$_id.building"},
		}},
		{Key: "rooms", Value: bson.D{
			{Key: "$push", Value: bson.D{
				{Key: "room", Value: "$_id.room"},
				{Key: "events", Value: "$events"},
			}},
		}},
	}}},
	{{Key: "$group", Value: bson.D{
		{Key: "_id", Value: bson.D{
			{Key: "date", Value: "$_id.date"},
		}},
		{Key: "buildings", Value: bson.D{
			{Key: "$push", Value: bson.D{
				{Key: "building", Value: "$_id.building"},
				{Key: "rooms", Value: "$rooms"},
			}},
		}},
	}}},
	//format date
	{{Key: "$project", Value: bson.D{
		{Key: "_id", Value: 0},
		{Key: "date", Value: bson.D{
			{Key: "$dateToString", Value: bson.D{
				{Key: "format", Value: "%Y-%m-%d"},
				{Key: "date", Value: "$_id.date"},
			}},
		}},
		{Key: "buildings", Value: 1},
	}}},
}
