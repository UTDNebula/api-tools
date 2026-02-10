/*
	This file contains the code for the research labs, centers, and facilities scraper.
*/

package scrapers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/api-tools/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ResearchListing represents a research lab, center, or facility
type ResearchListing struct {
	Id         primitive.ObjectID `json:"_id"`
	Name       string             `json:"name"`
	Link       string             `json:"link"`
	School     string             `json:"school,omitempty"`
	Professors []string           `json:"professors,omitempty"`
	Source     string             `json:"source"`
}

const facilitiesCentersURL = "https://www.utdallas.edu/research/facilities-centers/"
const labsURL = "https://labs.utdallas.edu/"

var professorsRegex = regexp.MustCompile(`\(([^)]+)\)`)

// ScrapeResearch scrapes research listings from UTD facilities-centers and labs pages
func ScrapeResearch(outDir string) {
	// NOTE (review feedback): This scraper is intentionally "best-effort" against the current public listing pages.
	// The upstream sources do not provide structured/complete research metadata (and may change without notice),
	// so we may need to rewrite/replace this when a better research data source becomes available.
	// Context: PR review discussion (Mike, Feb 2026) on issue #107 / PR #127.
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	client := newResearchHTTPClient()

	var listings []ResearchListing

	log.Println("Scraping research facilities and centers...")
	facilitiesListings := scrapeFacilitiesCenters(client)
	listings = append(listings, facilitiesListings...)
	log.Printf("Scraped %d facilities/centers", len(facilitiesListings))

	log.Println("Scraping research labs...")
	labsListings := scrapeLabs(client)
	listings = append(listings, labsListings...)
	log.Printf("Scraped %d labs", len(labsListings))

	listings = mergeResearchListings(listings)
	sort.Slice(listings, func(i, j int) bool {
		if listings[i].Name != listings[j].Name {
			return listings[i].Name < listings[j].Name
		}
		if listings[i].Link != listings[j].Link {
			return listings[i].Link < listings[j].Link
		}
		return listings[i].Source < listings[j].Source
	})

	log.Printf("Total research listings scraped: %d", len(listings))

	if err := utils.WriteJSON(fmt.Sprintf("%s/research.json", outDir), listings); err != nil {
		panic(err)
	}

	log.Printf("Successfully wrote research listings to %s/research.json", outDir)
}

func scrapeFacilitiesCenters(client *http.Client) []ResearchListing {
	doc := fetchDocumentWithRetry(client, facilitiesCentersURL)
	return parseFacilitiesCenters(doc)
}

func parseFacilitiesCenters(doc *goquery.Document) []ResearchListing {
	var listings []ResearchListing

	content := doc.Find("div.entry-content").First()
	var currentSection string
	var currentSchool string

	content.Find("h2.wp-block-heading, h3.wp-block-heading, ul.wp-block-list").Each(func(i int, s *goquery.Selection) {
		nodeName := goquery.NodeName(s)

		if nodeName == "h2" {
			currentSection = cleanText(s.Text())
			currentSchool = ""
			return
		}

		if nodeName == "h3" {
			// Skip "Related Pages"
			text := cleanText(s.Text())
			if text == "Related Pages" {
				// This section does not contain research listings.
				currentSection = ""
				currentSchool = ""
				return
			}
			currentSchool = text
			return
		}

		if nodeName == "ul" && currentSection != "" {
			s.Find("li").Each(func(j int, li *goquery.Selection) {
				name, link := parseLinkListItem(li, facilitiesCentersURL)
				school := cleanText(currentSchool)

				if name == "" {
					return
				}

				// Handle University-Wide section
				if currentSchool == "" && currentSection == "University-Wide Centers and Institutes" {
					school = "University-Wide"
				}

				listings = append(listings, ResearchListing{
					Id:     primitive.NewObjectID(),
					Name:   name,
					Link:   link,
					School: school,
					Source: "facilities-centers",
				})
			})
		}
	})

	return listings
}

func scrapeLabs(client *http.Client) []ResearchListing {
	doc := fetchDocumentWithRetry(client, labsURL)
	return parseLabs(doc)
}

func parseLabs(doc *goquery.Document) []ResearchListing {
	var listings []ResearchListing

	content := doc.Find("div.entry-content").First()
	var currentSchool string

	content.Find("h2.wp-block-heading, ul.wp-block-list").Each(func(i int, s *goquery.Selection) {
		nodeName := goquery.NodeName(s)

		if nodeName == "h2" {
			currentSchool = cleanText(s.Text())
			return
		}

		if nodeName == "ul" && currentSchool != "" {
			s.Find("li").Each(func(j int, li *goquery.Selection) {
				name, link := parseLinkListItem(li, labsURL)
				if name == "" {
					return
				}

				// Extract professors from parentheses
				afterText := cleanText(li.Clone().Children().Remove().End().Text())

				var professors []string
				if matches := professorsRegex.FindStringSubmatch(afterText); len(matches) > 1 {
					profText := matches[1]
					// Split by " and ", ", ", " & "
					profText = strings.ReplaceAll(profText, " and ", ",")
					profText = strings.ReplaceAll(profText, " & ", ",")
					parts := strings.Split(profText, ",")
					for _, p := range parts {
						p = cleanText(p)
						if p != "" {
							professors = append(professors, p)
						}
					}
					professors = uniqueStrings(professors)
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

func mergeResearchListings(listings []ResearchListing) []ResearchListing {
	byKey := make(map[string]ResearchListing, len(listings))
	for _, l := range listings {
		l.Name = cleanText(l.Name)
		l.School = cleanText(l.School)
		l.Link = cleanURL(l.Link, "")
		l.Source = cleanText(l.Source)
		l.Professors = uniqueStrings(l.Professors)

		key := strings.ToLower(l.Name) + "\n" + strings.ToLower(l.Link)
		if existing, ok := byKey[key]; ok {
			existing.School = mergeDelimited(existing.School, l.School, " | ")
			existing.Source = mergeDelimited(existing.Source, l.Source, " | ")
			existing.Professors = uniqueStrings(append(existing.Professors, l.Professors...))
			byKey[key] = existing
			continue
		}
		byKey[key] = l
	}

	merged := make([]ResearchListing, 0, len(byKey))
	for _, l := range byKey {
		merged = append(merged, l)
	}
	return merged
}

func mergeDelimited(a string, b string, delim string) string {
	a = cleanText(a)
	b = cleanText(b)
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a == b {
		return a
	}
	parts := uniqueStrings(strings.Split(a, delim))
	parts = uniqueStrings(append(parts, strings.Split(b, delim)...))
	sort.Strings(parts)
	return strings.Join(parts, delim)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = cleanText(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func parseLinkListItem(li *goquery.Selection, baseURL string) (string, string) {
	a := li.Find("a").First()
	if a.Length() == 0 {
		return cleanText(li.Text()), ""
	}

	name := cleanText(a.Text())
	link, _ := a.Attr("href")
	link = cleanURL(link, baseURL)
	return name, link
}

func cleanText(text string) string {
	text = strings.ReplaceAll(text, "\u00a0", " ")
	return strings.TrimSpace(text)
}

func cleanURL(raw string, base string) string {
	raw = cleanText(raw)
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	// Resolve relative links when we have a base.
	if base != "" && !u.IsAbs() {
		baseURL, baseErr := url.Parse(base)
		if baseErr == nil {
			u = baseURL.ResolveReference(u)
		}
	}

	u.Fragment = ""
	// Standardize trailing slash for a cleaner output (and better de-duping).
	if u.Path != "/" {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	return u.String()
}

func newResearchHTTPClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}
}

func fetchDocumentWithRetry(client *http.Client, pageURL string) *goquery.Document {
	var doc *goquery.Document
	delayedRetryCallback := func(numRetries int) {
		time.Sleep(250 * time.Millisecond * time.Duration(numRetries))
	}
	err := utils.Retry(func() error {
		d, err := fetchDocument(client, pageURL)
		if err != nil {
			return err
		}
		doc = d
		return nil
	}, 3, delayedRetryCallback)
	if err != nil {
		log.Panic(err)
	}
	return doc
}

func fetchDocument(client *http.Client, pageURL string) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, pageURL)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}
