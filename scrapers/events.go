package scrapers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const CALENDAR_LINK string = "https://calendar.utdallas.edu/calendar"

func ScrapeEvents(outDir string) {
	cancel := initChromeDp()
	defer cancel()

	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	events := []Event{}

	fmt.Printf("Scraping event page links\n")
	//Grab all links to event pages
	var pageLinks []string = []string{}
	_, err = chromedp.RunResponse(chromedpCtx,
		chromedp.Navigate(CALENDAR_LINK),
		chromedp.QueryAfter(".item.event_item.vevent > a",
			func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
				for _, node := range nodes {
					href, hasHref := node.Attribute("href")
					if !hasHref {
						return errors.New("event card was missing an href")
					}

					pageLinks = append(pageLinks, href)
				}
				return nil
			},
		),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Scraped event page links!\n")

	for _, page := range pageLinks {
		//Navigate to page and get page summary
		summary := ""
		_, err := chromedp.RunResponse(chromedpCtx,
			chromedp.Navigate(page),
			chromedp.QueryAfter(".summary",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					if !(len(nodes) == 0) {
						summary = getNodeText(nodes[0])
					}
					return nil
				}, chromedp.AtLeast(0),
			),
		)

		if err != nil {
			panic(err)
		}
		fmt.Printf("Navigated to page %s\n", summary)

		// Grab date/time of the event
		var dateTimeStart string = ""
		var dateTimeEnd string = ""
		err = chromedp.Run(chromedpCtx,
			chromedp.QueryAfter(".dtstart",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					if !(len(nodes) == 0) {
						time, hasTime := nodes[0].Attribute("title")
						if !hasTime {
							return errors.New("event does not have start time")
						}
						dateTimeStart = time
					}
					return nil
				}, chromedp.AtLeast(0),
			),
			chromedp.QueryAfter(".dtend",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					if !(len(nodes) == 0) {
						time, hasTime := nodes[0].Attribute("title")
						if !hasTime {
							return errors.New("event does not have end time")
						}
						dateTimeEnd = time
					}
					return nil
				}, chromedp.AtLeast(0),
			),
		)
		if err != nil {
			continue
		}
		fmt.Printf("Scraped time: %s to %s \n", dateTimeStart, dateTimeEnd)

		//Grab Location of Event
		var location string = ""
		err = chromedp.Run(chromedpCtx,
			chromedp.QueryAfter("p.location > span",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					if !(len(nodes) == 0) {
						location = getNodeText(nodes[0])
					}
					return nil
				}, chromedp.AtLeast(0),
			),
		)
		if err != nil {
			continue
		}
		fmt.Printf("Scraped location: %s, \n", location)

		//Get description of event
		var description string = ""
		err = chromedp.Run(chromedpCtx,
			chromedp.QueryAfter(".description > p",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					if !(len(nodes) == 0) {
						description = getNodeText(nodes[0])
					}
					return nil
				}, chromedp.AtLeast(0),
			),
		)
		if err != nil {
			continue
		}
		fmt.Printf("Scraped description: %s, \n", description)

		//Grab Event Type
		var eventType []string = []string{}
		err = chromedp.Run(chromedpCtx,
			chromedp.QueryAfter(".filter-event_types > p > a",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					for _, node := range nodes {
						eventType = append(eventType, getNodeText(node))
					}
					return nil
				}, chromedp.AtLeast(0),
			),
		)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Scraped event type: %s\n", eventType)

		//Grab Target Audience
		targetAudience := []string{}
		err = chromedp.Run(chromedpCtx,
			chromedp.QueryAfter(".filter-event_target_audience > p > a",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					for _, node := range nodes {
						targetAudience = append(targetAudience, getNodeText(node))
					}
					return nil
				}, chromedp.AtLeast(0),
			),
		)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Scraped target audience: %s, \n", targetAudience)

		//Grab Topic
		topic := []string{}
		err = chromedp.Run(chromedpCtx,
			chromedp.QueryAfter(".filter-event_topic > p > a",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					for _, node := range nodes {
						topic = append(topic, getNodeText(node))
					}
					return nil
				}, chromedp.AtLeast(0),
			),
		)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Scraped topic: %s, \n", topic)

		//Grab Event Tags
		tags := []string{}
		err = chromedp.Run(chromedpCtx,
			chromedp.QueryAfter(".event-tags > p > a",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					for _, node := range nodes {
						tags = append(tags, getNodeText(node))
					}
					return nil
				}, chromedp.AtLeast(0),
			),
		)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Scraped tags: %s, \n", tags)

		//Grab Website
		var eventWebsite string = ""
		err = chromedp.Run(chromedpCtx,
			chromedp.QueryAfter(".event-website > p > a",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					if !(len(nodes) == 0) {
						href, hasHref := nodes[0].Attribute("href")
						if !hasHref {
							return errors.New("event does not have website")
						}
						eventWebsite = href
					}
					return nil
				}, chromedp.AtLeast(0),
			),
		)
		if err != nil {
			continue
		}
		fmt.Printf("Scraped website: %s, \n", eventWebsite)

		//Grab Department
		var eventDepartment []string = []string{}
		err = chromedp.Run(chromedpCtx,
			chromedp.QueryAfter(".event-group > a",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					for _, node := range nodes {
						eventDepartment = append(eventDepartment, getNodeText(node))
					}
					return nil
				}, chromedp.AtLeast(0),
			),
		)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Scraped department: %s, \n", eventDepartment)

		//Grab Contact information
		var contactInformationName string = ""
		var contactInformationEmail string = ""
		var contactInformationPhone string = ""
		err = chromedp.Run(chromedpCtx,
			chromedp.QueryAfter(".custom-field-contact_information_name",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					if !(len(nodes) == 0) {
						contactInformationName = getNodeText(nodes[0])
					}
					return nil
				}, chromedp.AtLeast(0),
			),
			chromedp.QueryAfter(".custom-field-contact_information_email",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					if !(len(nodes) == 0) {
						contactInformationEmail = getNodeText(nodes[0])
					}
					return nil
				}, chromedp.AtLeast(0),
			),
			chromedp.QueryAfter(".custom-field-contact_information_phone",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					if !(len(nodes) == 0) {
						contactInformationPhone = getNodeText(nodes[0])
						if err != nil {
							return err
						}
					}
					return nil
				}, chromedp.AtLeast(0),
			),
		)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Scraped contact name info: %s\n", contactInformationName)
		fmt.Printf("Scraped contact email info: %s\n", contactInformationEmail)
		fmt.Printf("Scraped contact phone info: %s\n", contactInformationPhone)

		events = append(events, Event{
			//Id:                 schema.IdWrapper{Id: primitive.NewObjectID()},
			Summary:            summary,
			Location:           location,
			StartTime:          dateTimeStart,
			EndTime:            dateTimeEnd,
			Description:        description,
			EventType:          eventType,
			TargetAudience:     targetAudience,
			Topic:              topic,
			EventTags:          tags,
			EventWebsite:       eventWebsite,
			Department:         eventDepartment,
			ContactName:        contactInformationName,
			ContactEmail:       contactInformationEmail,
			ContactPhoneNumber: contactInformationPhone,
		})

		// Write event data to output file
		fptr, err := os.Create(fmt.Sprintf("%s/Events.json", outDir))
		if err != nil {
			panic(err)
		}
		encoder := json.NewEncoder(fptr)
		encoder.SetIndent("", "\t")
		encoder.Encode(events)
		fptr.Close()
	}
}

//Temporary until chanes to nebula-api schema get merged
type Event struct {
	Id                 primitive.ObjectID `bson:"_id" json:"_id"`
	Summary            string             `bson:"summary" json:"summary"`
	Location           string             `bson:"location" json:"location"`
	StartTime          string             `bson:"start_time" json:"start_time"`
	EndTime            string             `bson:"end_time" json:"end_time"`
	Description        string             `bson:"description" json:"description"`
	EventType          []string           `bson:"event_type" json:"event_type"`
	TargetAudience     []string           `bson:"target_audience" json:"target_audience"`
	Topic              []string           `bson:"topic" json:"topic"`
	EventTags          []string           `bson:"event_tags" json:"event_tags"`
	EventWebsite       string             `bson:"event_website" json:"event_website"`
	Department         []string           `bson:"department" json:"department"`
	ContactName        string             `bson:"contact_name" json:"contact_name"`
	ContactEmail       string             `bson:"contact_email" json:"contact_email"`
	ContactPhoneNumber string             `bson:"contact_phone_number" json:"contact_phone_number"`
}
