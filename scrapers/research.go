package scrapers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ResearchListing represents a research lab, center, or facility
type ResearchListing struct {
	Id          primitive.ObjectID `json:"_id"`
	Name        string             `json:"name"`
	Link        string             `json:"link"`
	School      string             `json:"school,omitempty"`
	Professors  []string           `json:"professors,omitempty"`
	Source      string             `json:"source"`
}

const facilitiesCentersURL = "https://www.utdallas.edu/research/facilities-centers/"
const labsURL = "https://labs.utdallas.edu/"

var professorsRegex = regexp.MustCompile(`\(([^)]+)\)`)

// ScrapeResearch scrapes research listings from UTD facilities-centers and labs pages
func ScrapeResearch(outDir string) {
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	var listings []ResearchListing

	log.Println("Scraping research facilities and centers...")
	facilitiesListings := scrapeFacilitiesCenters()
	listings = append(listings, facilitiesListings...)
	log.Printf("Scraped %d facilities/centers", len(facilitiesListings))

	log.Println("Scraping research labs...")
	labsListings := scrapeLabs()
	listings = append(listings, labsListings...)
	log.Printf("Scraped %d labs", len(labsListings))

	log.Printf("Total research listings scraped: %d", len(listings))

	// Write to JSON
	outPath := fmt.Sprintf("%s/research.json", outDir)
	fptr, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer fptr.Close()

	encoder := json.NewEncoder(fptr)
	encoder.SetIndent("", "\t")
	if err := encoder.Encode(listings); err != nil {
		panic(err)
	}

	log.Printf("Successfully wrote research listings to %s", outPath)
}

func scrapeFacilitiesCenters() []ResearchListing {
	doc := fetchDocument(facilitiesCentersURL)
	var listings []ResearchListing

	content := doc.Find("div.entry-content").First()
	var currentSection string
	var currentSchool string

	content.Find("h2.wp-block-heading, h3.wp-block-heading, ul.wp-block-list").Each(func(i int, s *goquery.Selection) {
		nodeName := goquery.NodeName(s)

		if nodeName == "h2" {
			currentSection = strings.TrimSpace(s.Text())
			currentSchool = ""
			return
		}

		if nodeName == "h3" {
			// Skip "Related Pages"
			text := strings.TrimSpace(s.Text())
			if text == "Related Pages" {
				return
			}
			currentSchool = text
			return
		}

		if nodeName == "ul" && currentSection != "" {
			s.Find("li").Each(func(j int, li *goquery.Selection) {
				name := ""
				link := ""
				school := currentSchool

				// Try to find an anchor
				a := li.Find("a").First()
				if a.Length() > 0 {
					name = strings.TrimSpace(a.Text())
					link, _ = a.Attr("href")
				}

				// If no anchor or anchor is not the primary content, use full text
				fullText := strings.TrimSpace(li.Text())
				if name == "" || (fullText != name && !strings.HasPrefix(fullText, name)) {
					// Extract name from full text (everything before the anchor if embedded)
					if name != "" && strings.Contains(fullText, name) {
						// Name is embedded, extract prefix
						idx := strings.Index(fullText, name)
						if idx > 0 {
							name = strings.TrimSpace(fullText[:idx])
						} else {
							name = fullText
						}
					} else {
						name = fullText
					}
				}

				if name == "" {
					return
				}

				// Handle University-Wide section
				if currentSchool == "" && currentSection == "University-Wide Centers and Institutes" {
					school = "University-Wide"
				}

				listings = append(listings, ResearchListing{
					Id:         primitive.NewObjectID(),
					Name:       name,
					Link:       link,
					School:     school,
					Professors: []string{},
					Source:     "facilities-centers",
				})
			})
		}
	})

	return listings
}

func scrapeLabs() []ResearchListing {
	doc := fetchDocument(labsURL)
	var listings []ResearchListing

	content := doc.Find("div.entry-content").First()
	var currentSchool string

	content.Find("h2.wp-block-heading, ul.wp-block-list").Each(func(i int, s *goquery.Selection) {
		nodeName := goquery.NodeName(s)

		if nodeName == "h2" {
			currentSchool = strings.TrimSpace(s.Text())
			return
		}

		if nodeName == "ul" && currentSchool != "" {
			s.Find("li").Each(func(j int, li *goquery.Selection) {
				a := li.Find("a").First()
				if a.Length() == 0 {
					return
				}

				name := strings.TrimSpace(a.Text())
				link, _ := a.Attr("href")

				// Extract professors from parentheses
				afterText := strings.TrimSpace(li.Clone().Children().Remove().End().Text())

				var professors []string
				if matches := professorsRegex.FindStringSubmatch(afterText); len(matches) > 1 {
					profText := matches[1]
					// Split by " and ", ", ", " & "
					profText = strings.ReplaceAll(profText, " and ", ",")
					profText = strings.ReplaceAll(profText, " & ", ",")
					parts := strings.Split(profText, ",")
					for _, p := range parts {
						p = strings.TrimSpace(p)
						if p != "" {
							professors = append(professors, p)
						}
					}
				}

				listings = append(listings, ResearchListing{
					Id:         primitive.NewObjectID(),
					Name:       name,
					Link:       link,
					School:     currentSchool,
					Professors: professors,
					Source:     "labs",
				})
			})
		}
	})

	return listings
}

func fetchDocument(url string) *goquery.Document {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		panic(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, url))
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		panic(err)
	}

	return doc
}
