package parser

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TestParseDiscountItem tests parsing of individual discount entries
func TestParseDiscountItem(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		html     string
		category string
		expected schema.DiscountProgram
	}{
		"complete_entry": {
			html: `<div class="container cditem">
				<div class="row">
					<div class="col-sm">
						<p class="h5"><a href="https://www.airbnb.com/">Airbnb Houses Near UTD</a></p>
						<p></p>
						Tim Bao<br>
						972-214-5510<br>
						<p><a href="mailto:timmy.bao@gmail.com">timmy.bao@gmail.com</a></p>
					</div>
					<div class="col-sm">
						<p class="pt-3"></p><p>10% discount to any Comet Card holder from UTD.</p>
						<p></p>
					</div>
				</div>
			</div>`,
			category: "Accommodations",
			expected: schema.DiscountProgram{
				Category: "Accommodations",
				Business: "Airbnb Houses Near UTD",
				Address:  "",
				Phone:    "972-214-5510",
				Email:    "timmy.bao@gmail.com",
				Website:  "https://www.airbnb.com/",
				Discount: "10% discount to any Comet Card holder from UTD.",
			},
		},
		"with_address": {
			html: `<div class="container cditem">
				<div class="row">
					<div class="col-sm">
						<p class="h5"><a href="http://www.marriott.com/daler">Element Dallas Richardson</a></p>
						<p></p><p>2205 N. Glenville Drive, Richardson, Texas 75082</p>
						<p></p>
						Jennifer Howard<br>
						972.833.1771<br>
						<p><a href="mailto:jlhoward@elementdallasrichardson.com">jlhoward@elementdallasrichardson.com</a></p>
					</div>
					<div class="col-sm">
						<p class="pt-3"></p><p>Receive up to 25% off retail rates by using UTD promo code – UTX</p>
						<p></p>
					</div>
				</div>
			</div>`,
			category: "Accommodations",
			expected: schema.DiscountProgram{
				Category: "Accommodations",
				Business: "Element Dallas Richardson",
				Address:  "2205 N. Glenville Drive, Richardson, Texas 75082",
				Phone:    "972.833.1771",
				Email:    "jlhoward@elementdallasrichardson.com",
				Website:  "http://www.marriott.com/daler",
				Discount: "Receive up to 25% off retail rates by using UTD promo code – UTX",
			},
		},
		"no_link": {
			html: `<div class="container cditem">
				<div class="row">
					<div class="col-sm">
						<p class="h5">MasterTech</p>
						<p></p><p>1300 Alma Dr. Plano, Tx.</p>
						<p></p>
						Bill Mertz<br>
						972-578-1841<br>
						<p><a href="mailto:Bill.mastertech@gmail.com">Bill.mastertech@gmail.com</a></p>
					</div>
					<div class="col-sm">
						<p class="pt-3"></p><p>10% off both parts and labor up to $150 off (excluding sublet).</p>
						<p></p>
					</div>
				</div>
			</div>`,
			category: "Auto Services",
			expected: schema.DiscountProgram{
				Category: "Auto Services",
				Business: "MasterTech",
				Address:  "1300 Alma Dr. Plano, Tx.",
				Phone:    "972-578-1841",
				Email:    "Bill.mastertech@gmail.com",
				Website:  "",
				Discount: "10% off both parts and labor up to $150 off (excluding sublet).",
			},
		},
		"html_entities": {
			html: `<div class="container cditem">
				<div class="row">
					<div class="col-sm">
						<p class="h5"><a href="http://test.com">J&amp;S Party Rental</a></p>
						<p></p><p>4906 Dillehay Dr. #300 Allen, TX 75002</p>
						<p></p>
						<p><a href="mailto:admin@test.com">admin@test.com</a></p>
					</div>
					<div class="col-sm">
						<p class="pt-3"></p><p>We&#39;re your one-stop shop &amp; more.</p>
						<p></p>
					</div>
				</div>
			</div>`,
			category: "Entertainment",
			expected: schema.DiscountProgram{
				Category: "Entertainment",
				Business: "J&S Party Rental",
				Address:  "4906 Dillehay Dr. #300 Allen, TX 75002",
				Phone:    "",
				Email:    "admin@test.com",
				Website:  "http://test.com",
				Discount: "We're your one-stop shop & more.",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("failed to parse HTML: %v", err)
			}

			result := parseDiscountItem(doc.Find("div.cditem").First(), tc.category)
			if result == nil {
				t.Fatal("parseDiscountItem returned nil")
			}

			diff := cmp.Diff(tc.expected, *result,
				cmpopts.IgnoreFields(schema.DiscountProgram{}, "Id"),
			)

			if diff != "" {
				t.Errorf("parseDiscountItem() mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

// TestIsValidDiscount tests the discount validation logic
func TestIsValidDiscount(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		discount *schema.DiscountProgram
		expected bool
	}{
		"valid_complete": {
			discount: &schema.DiscountProgram{
				Business: "Test Business",
				Discount: "10% off",
				Email:    "test@example.com",
			},
			expected: true,
		},
		"valid_minimal": {
			discount: &schema.DiscountProgram{
				Business: "Test Business",
				Website:  "https://example.com",
			},
			expected: true,
		},
		"invalid_no_business": {
			discount: &schema.DiscountProgram{
				Business: "",
				Discount: "10% off",
			},
			expected: false,
		},
		"invalid_business_name": {
			discount: &schema.DiscountProgram{
				Business: "Business",
				Discount: "10% off",
			},
			expected: false,
		},
		"invalid_no_content": {
			discount: &schema.DiscountProgram{
				Business: "Test Business",
			},
			expected: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := isValidDiscount(tc.discount)
			if result != tc.expected {
				t.Errorf("isValidDiscount() = %v, expected %v", result, tc.expected)
			}
		})
	}
}

// TestCleanText tests HTML entity decoding and whitespace trimming
func TestCleanText(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input    string
		expected string
	}{
		"ampersand": {
			input:    "J&amp;S Party Rental",
			expected: "J&S Party Rental",
		},
		"apostrophe": {
			input:    "We&#39;re the best",
			expected: "We're the best",
		},
		"multiple_entities": {
			input:    "&lt;div&gt; Test &amp; More &lt;/div&gt;",
			expected: "<div> Test & More </div>",
		},
		"whitespace": {
			input:    "  Test  Business  ",
			expected: "Test  Business",
		},
		"newlines": {
			input:    "Test\nBusiness\n",
			expected: "Test\nBusiness",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := cleanText(tc.input)
			if result != tc.expected {
				t.Errorf("cleanText(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestContainsPhonePattern tests phone number pattern detection
func TestContainsPhonePattern(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input    string
		expected bool
	}{
		"standard": {
			input:    "972-214-5510",
			expected: true,
		},
		"parentheses": {
			input:    "(972) 214-5510",
			expected: true,
		},
		"not_phone": {
			input:    "Hello World",
			expected: false,
		},
		"single_dash": {
			input:    "Test-Name",
			expected: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := containsPhonePattern(tc.input)
			if result != tc.expected {
				t.Errorf("containsPhonePattern(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestIsNumericPhone tests numeric phone detection
func TestIsNumericPhone(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input    string
		expected bool
	}{
		"numeric_phone": {
			input:    "9722145510",
			expected: true,
		},
		"with_spaces": {
			input:    "972 214 5510",
			expected: true,
		},
		"too_short": {
			input:    "12345",
			expected: false,
		},
		"too_long": {
			input:    "123456789012345678901",
			expected: false,
		},
		"not_numeric": {
			input:    "Hello World",
			expected: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := isNumericPhone(tc.input)
			if result != tc.expected {
				t.Errorf("isNumericPhone(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestExtractEmail tests email extraction from text
func TestExtractEmail(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input    string
		expected string
	}{
		"simple": {
			input:    "test@example.com",
			expected: "test@example.com",
		},
		"with_text": {
			input:    "Contact us at hello@company.com for more info",
			expected: "hello@company.com",
		},
		"no_email": {
			input:    "No email here",
			expected: "No email here",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := extractEmail(tc.input)
			if result != tc.expected {
				t.Errorf("extractEmail(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestTrimAfter tests substring extraction after a separator
func TestTrimAfter(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input    string
		sep      string
		expected string
	}{
		"mailto": {
			input:    "mailto:test@example.com",
			sep:      "mailto:",
			expected: "test@example.com",
		},
		"not_found": {
			input:    "test@example.com",
			sep:      "mailto:",
			expected: "test@example.com",
		},
		"middle": {
			input:    "prefix::value",
			sep:      "::",
			expected: "value",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := trimAfter(tc.input, tc.sep)
			if result != tc.expected {
				t.Errorf("trimAfter(%q, %q) = %q, expected %q", tc.input, tc.sep, result, tc.expected)
			}
		})
	}
}
