package utils

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

// TestMain loads environment variables for utils package tests.
func TestMain(m *testing.M) {
	// Load .env vars for testing
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// The following environment variables must be set for tests to run
	envVars := []string{
		"LOGIN_NETID", "LOGIN_PASSWORD", "LOGIN_ASTRA_USERNAME", "LOGIN_ASTRA_PASSWORD",
	}

	for _, envVar := range envVars {
		// Get each env, and if it doesn't exist, error
		v, err := GetEnv(envVar)
		if v == "" || err != nil {
			log.Fatalf("Error loading environment variable %v: %v\n"+
				"Valid credentials are required for testing utils/methods", envVar, err)
		}
	}

	// Run tests
	os.Exit(m.Run())
}

func TestInitChromeDP(t *testing.T) {
	// Test to run in both headless and non-headless
	test := func(t *testing.T) {
		ctx, cancel := InitChromeDp()
		defer cancel()
		if ctx.Err() != nil {
			t.Errorf("Failed to initialize chromedp: %v", ctx.Err())
		}
	}

	Headless = false
	t.Run("not-headless", test)

	Headless = true
	t.Run("headless", test)

}

// TestRefreshToken confirms coursebook tokens refresh under both headless settings.
func TestRefreshToken(t *testing.T) {
	// Get a chromedp context
	ctx, cancel := InitChromeDp()
	defer cancel()
	// Try refreshing token
	headers := RefreshToken(ctx)

	// Make sure we successfully got a PTGSESSID cookie
	// Check to make sure we have a cookie
	found := false
	for _, cookie := range headers["Cookie"] {
		if strings.HasPrefix(cookie, "PTGSESSID") {
			found = true
		}
	}

	// Fail if no PTGSESSID cookie found
	if !found {
		t.Errorf("Failed to get PTGSESSID cookie from RefreshToken!")
	}
}

// TestRefreshAstraToken test RefreshAstraToken returns the proper cookie
func TestRefreshAstraToken(t *testing.T) {
	ctx, cancel := InitChromeDp()
	defer cancel()
	// Try refreshing token
	headers := RefreshAstraToken(ctx)

	// Make sure we successfully got a PTGSESSID cookie
	// Check to make sure we have a cookie
	found := false
	// Split cookie string to get each cookie
	cookies := strings.Split(headers["Cookie"][0], "; ")
	for _, cookie := range cookies {
		if strings.HasPrefix(cookie, "UTXDallas.ASPXFORMSAUTH") {
			found = true
		}
	}

	// Fail if cookie not found
	if !found {
		t.Errorf("Failed to get UTXDallas.ASPXFORMSAUTH cookie from RefreshAstraToken!")
	}
}

// TestGetCoursePrefixes tests GetCoursePrefixes at least 20 course prefixes
func TestGetCoursePrefixes(t *testing.T) {
	ctx, cancel := InitChromeDp()
	defer cancel()
	prefixes := GetCoursePrefixes(ctx)
	t.Log(len(prefixes))

	// There should be at least 20 course prefixes
	if len(prefixes) < 20 {
		t.Errorf("Failed to fetch at least 20 course prefixes. Found %d", len(prefixes))
	}
}
