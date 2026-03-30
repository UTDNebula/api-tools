package parser

import (
	"fmt"
	"html"
	"log"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/dongri/phonenumber"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ParseDiscounts reads the scraped discount HTML and produces structured discount JSON.
func ParseDiscounts(inDir string, outDir string) {
	// Read the scraped HTML file
	htmlPath := fmt.Sprintf("%s/discountsScraped.html", inDir)
	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		panic(err)
	}

	log.Println("Parsing discount entries...")

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlBytes)))
	if err != nil {
		panic(err)
	}

	// Find the main content area
	content := doc.Find("article .entry-content").First()
	if content.Length() == 0 {
		panic("failed to find content area")
	}

	var discounts []schema.DiscountProgram
	var currentCategory string

	// Find all discount items - they're in div.cditem containers
	content.Find("h3.cdpview, div.cditem").Each(func(i int, s *goquery.Selection) {
		// Check if this is a category header
		if s.Is("h3.cdpview") {
			currentCategory = strings.TrimSpace(s.Text())
			return
		}

		// This is a discount entry
		discount := parseDiscountItem(s, currentCategory)
		if discount != nil && isValidDiscount(discount) {
			discounts = append(discounts, *discount)
		}
	})

	if err := utils.WriteJSON(fmt.Sprintf("%s/discounts.json", outDir), discounts); err != nil {
		panic(err)
	}

	log.Printf("Parsed %d discount programs successfully!", len(discounts))
}

// parseDiscountItem extracts discount information from a cditem div
func parseDiscountItem(s *goquery.Selection, category string) *schema.DiscountProgram {
	discount := &schema.DiscountProgram{
		Id:       primitive.NewObjectID(),
		Category: category,
	}

	// The structure has two columns: business info and discount info
	cols := s.Find("div.col-sm")
	if cols.Length() != 2 {
		return nil
	}

	// First column: business info
	businessCol := cols.Eq(0)

	// Get business name from p.h5
	businessName := businessCol.Find("p.h5").First()
	if businessName.Length() > 0 {
		// Try to get link text first, otherwise plain text
		link := businessName.Find("a").First()
		if link.Length() > 0 {
			discount.Business = cleanText(link.Text())
			if href, exists := link.Attr("href"); exists {
				discount.Website = href
			}
		} else {
			discount.Business = cleanText(businessName.Text())
		}
	}

	// Extract address, phone, email from remaining paragraphs
	var addressLines []string
	businessCol.Find("p").Each(func(j int, p *goquery.Selection) {
		// Skip the business name paragraph
		if p.HasClass("h5") {
			return
		}

		text := cleanText(p.Text())
		if text == "" {
			return
		}

		// Check for email
		emailLink := p.Find("a[href^='mailto:']").First()
		if emailLink.Length() > 0 {
			if href, exists := emailLink.Attr("href"); exists {
				discount.Email = trimAfter(href, "mailto:")
			}
		} else if strings.Contains(text, "@") {
			discount.Email = extractEmail(text)
		}

		// If it's not email and doesn't look like a single name, treat as address
		if !strings.Contains(text, "@") && len(text) > 10 {
			addressLines = append(addressLines, text)
		}
	})

	// Extract phone from text nodes (they're often br-separated, not in p tags)
	businessHTML, _ := businessCol.Html()
	lines := strings.Split(businessHTML, "<br")
	for _, line := range lines {
		// Strip HTML tags
		line = stripHTMLTags(line)
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if it's a phone number
		// phonenumber.Parse returns "" if not parsable as a phone number
		// assumes that all phone numbers here are US phone numbers
		parsed := phonenumber.Parse(line, "US")
		if parsed != "" {
			discount.Phone = parsed
		}

	}

	var addresses = []string{}
	for i := 0; i < len(addressLines); i++ {
		currentLine := addressLines[i]

		// Cleaning up the line
		currentLine = strings.ReplaceAll(currentLine, "\n", " ")
		currentLine = strings.ReplaceAll(currentLine, "\r", " ")

		addresses = append(addresses, currentLine)
	}

	discount.Address = addresses

	// Second column: discount info
	discountCol := cols.Eq(1)
	var discountTexts []string
	discountCol.Find("p").Each(func(j int, p *goquery.Selection) {
		text := cleanText(p.Text())
		if text != "" && !strings.HasPrefix(text, "pt-") {
			discountTexts = append(discountTexts, text)
		}
	})

	// Join discount texts and keep newlines for multi-paragraph descriptions
	discount.Discount = strings.Join(discountTexts, "\n")

	return discount
}

// cleanText removes HTML entities and extra whitespace
func cleanText(s string) string {
	// Decode HTML entities like &amp; to &
	s = html.UnescapeString(s)
	// Trim whitespace
	s = strings.TrimSpace(s)
	return s
}

// stripHTMLTags removes HTML tags from a string
func stripHTMLTags(s string) string {
	// Simple regex to remove HTML tags
	s = strings.ReplaceAll(s, "/>", "")
	s = strings.ReplaceAll(s, ">", "")
	idx := strings.Index(s, "<")
	if idx >= 0 {
		s = s[:idx]
	}
	return s
}

// isValidDiscount checks if a discount entry has meaningful data
func isValidDiscount(d *schema.DiscountProgram) bool {
	// Must have a business name
	if d.Business == "" {
		return false
	}

	// Filter out obvious non-businesses
	businessLower := strings.ToLower(d.Business)
	invalidNames := []string{"business", "discount", "categories", "vendors", "contact"}
	if slices.Contains(invalidNames, businessLower) {
		return false
	}

	// Must have at least a discount or some contact info
	hasContent := d.Discount != "" || d.Email != "" || d.Phone != "" ||
		d.Website != "" || len(d.Address) > 0

	return hasContent
}

// extractEmail uses regex to extract email addresses from text
func extractEmail(text string) string {
	const emailRegexPattern = `[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,4}`
	var emailRegex = regexp.MustCompile(emailRegexPattern)

	email := emailRegex.FindString(text)

	if email == "" {
		return "No email here"
	}

	return email
}

// trimAfter returns the substring after the first occurrence of sep
func trimAfter(s, sep string) string {
	if _, after, ok := strings.Cut(s, sep); ok {
		return after
	}
	return s
}
