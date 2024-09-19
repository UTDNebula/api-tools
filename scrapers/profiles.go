/*
	This file contains the code for the professor profile scraper.
*/

package scrapers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const BASE_URL string = "https://profiles.utdallas.edu/browse?page="

var primaryLocationRegex *regexp.Regexp = regexp.MustCompile(`^(\w+)\s+(\d+\.\d{3}[A-z]?)$`)
var fallbackLocationRegex *regexp.Regexp = regexp.MustCompile(`^([A-z]+)(\d+)\.?(\d{3}[A-z]?)$`)

func parseLocation(text string) schema.Location {
	var building string
	var room string

	submatches := primaryLocationRegex.FindStringSubmatch(text)
	if submatches == nil {
		submatches = fallbackLocationRegex.FindStringSubmatch(text)
		if submatches == nil {
			return schema.Location{}
		} else {
			building = submatches[1]
			room = fmt.Sprintf("%s.%s", submatches[2], submatches[3])
		}
	} else {
		building = submatches[1]
		room = submatches[2]
	}

	return schema.Location{
		Building: building,
		Room:     room,
		Map_uri:  fmt.Sprintf("https://locator.utdallas.edu/%s_%s", building, room),
	}
}

func parseList(list []string) (string, schema.Location) {
	var phoneNumber string
	var office schema.Location

	for _, element := range list {
		element = strings.TrimSpace(element)
		utils.VPrintf("Element is: %s", element)
		if strings.Contains(element, "-") {
			phoneNumber = element
		} else if primaryLocationRegex.MatchString(element) || fallbackLocationRegex.MatchString(element) {
			utils.VPrintf("Element match is: %s", element)
			office = parseLocation(element)
			break
		}
	}

	return phoneNumber, office
}

func parseName(fullName string) (string, string) {
	commaIndex := strings.Index(fullName, ",")
	if commaIndex != -1 {
		fullName = fullName[:commaIndex]
	}
	names := strings.Split(fullName, " ")
	ultimateName := strings.ToLower(names[len(names)-1])
	if len(names) > 2 && (ultimateName == "jr" ||
		ultimateName == "sr" ||
		ultimateName == "I" ||
		ultimateName == "II" ||
		ultimateName == "III") {
		names = names[:len(names)-1]
	}
	return names[0], names[len(names)-1]
}

func getNodeText(node *cdp.Node) string {
	if len(node.Children) == 0 {
		return ""
	}
	return node.Children[0].NodeValue
}

func scrapeProfessorLinks(chromedpCtx context.Context) []string {
	var pageLinks []*cdp.Node
	_, err := chromedp.RunResponse(chromedpCtx,
		chromedp.Navigate(BASE_URL+"1"),
		chromedp.QueryAfter(".page-link",
			func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
				pageLinks = nodes
				return nil
			},
		),
	)
	if err != nil {
		panic(err)
	}

	numPages, err := strconv.Atoi(getNodeText(pageLinks[len(pageLinks)-2]))
	if err != nil {
		panic(err)
	}

	professorLinks := make([]string, 0, numPages)
	for curPage := 1; curPage <= numPages; curPage++ {
		_, err := chromedp.RunResponse(chromedpCtx,
			chromedp.Navigate(BASE_URL+strconv.Itoa(curPage)),
			chromedp.QueryAfter("//h5[@class='card-title profile-name']//a",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					for _, node := range nodes {
						href, hasHref := node.Attribute("href")
						if !hasHref {
							return errors.New("professor card was missing an href")
						}
						professorLinks = append(professorLinks, href)
					}
					return nil
				},
			),
		)
		if err != nil {
			panic(err)
		}
	}

	return professorLinks
}

func ScrapeProfiles(outDir string) {

	chromedpCtx, cancel := utils.InitChromeDp()
	defer cancel()

	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	var professors []schema.Professor

	log.Print("Scraping professor links...")
	professorLinks := scrapeProfessorLinks(chromedpCtx)
	log.Print("Scraped professor links!")

	for _, link := range professorLinks {

		// Navigate to the link and get the names
		var firstName, lastName string

		utils.VPrint("Scraping name...")

		_, err := chromedp.RunResponse(chromedpCtx,
			chromedp.Navigate(link),
			chromedp.ActionFunc(func(ctx context.Context) error {
				var text string
				err := chromedp.Text("div.contact_info>h1", &text).Do(ctx)
				firstName, lastName = parseName(text)
				return err
			}),
		)
		if err != nil {
			panic(err)
		}

		// Get the image uri
		var imageUri string

		utils.VPrint("Scraping imageUri...")

		err = chromedp.Run(chromedpCtx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				var attributes map[string]string
				err := chromedp.Attributes("//img[@class='profile_photo']", &attributes, chromedp.AtLeast(0)).Do(ctx)
				if err == nil {
					var hasSrc bool
					imageUri, hasSrc = attributes["src"]
					if !hasSrc {
						return errors.New("no src found for imageUri")
					}
				}
				return err
			}),
		)
		if err != nil {
			err = chromedp.Run(chromedpCtx,
				chromedp.ActionFunc(func(ctx context.Context) error {
					var attributes map[string]string
					err := chromedp.Attributes("//div[@class='profile-header  fancy_header ']", &attributes, chromedp.AtLeast(0)).Do(ctx)
					if err == nil {
						var hasStyle bool
						imageUri, hasStyle = attributes["style"]
						if !hasStyle {
							return errors.New("no style found for imageUri")
						}
						imageUri = imageUri[23 : len(imageUri)-3]
					}
					return err
				}),
			)
			if err != nil {
				panic(err)
			}
		}

		// Get the titles
		titles := make([]string, 0, 3)

		utils.VPrint("Scraping titles...")

		err = chromedp.Run(chromedpCtx,
			chromedp.QueryAfter("div.profile-title",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					for _, node := range nodes {
						tempText := getNodeText(node)
						if !strings.Contains(tempText, "$") {
							titles = append(titles, tempText)
						}
					}
					return nil
				}, chromedp.AtLeast(0),
			),
		)
		if err != nil {
			continue
		}

		// Get the email
		var email string

		utils.VPrint("Scraping email...")

		err = chromedp.Run(chromedpCtx,
			chromedp.Text("//a[contains(@id,'☄️')]", &email, chromedp.AtLeast(0)),
		)
		if err != nil {
			continue
		}

		// Get the phone number and office location
		var texts []string

		utils.VPrint("Scraping list text...")

		err = chromedp.Run(chromedpCtx,
			chromedp.QueryAfter("div.contact_info>div ~ div",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					var tempText string
					err := chromedp.Text("div.contact_info>div ~ div", &tempText).Do(ctx)
					texts = strings.Split(tempText, "\n")
					return err
				},
			),
		)
		if err != nil {
			panic(err)
		}

		utils.VPrint("Parsing list...")
		phoneNumber, office := parseList(texts)
		utils.VPrintf("Parsed list! #: %s, Office: %v", phoneNumber, office)

		professors = append(professors, schema.Professor{
			Id:           primitive.NewObjectID(),
			First_name:   firstName,
			Last_name:    lastName,
			Titles:       titles,
			Email:        email,
			Phone_number: phoneNumber,
			Office:       office,
			Profile_uri:  link,
			Image_uri:    imageUri,
			Office_hours: []schema.Meeting{},
			Sections:     []primitive.ObjectID{},
		})

		utils.VPrintf("Scraped profile for %s %s!", firstName, lastName)
	}

	// Write professor data to output file
	fptr, err := os.Create(fmt.Sprintf("%s/profiles.json", outDir))
	if err != nil {
		panic(err)
	}
	encoder := json.NewEncoder(fptr)
	encoder.SetIndent("", "\t")
	encoder.Encode(professors)
	fptr.Close()
}
