package utils

import (
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

// TestMain loads environment variables for utils package tests.
func TestMain(m *testing.M) {
	// Load .env vars for testing
	godotenv.Load("../.env")
	// Run tests
	os.Exit(m.Run())
}

// TestInitChromeDp ensures chromedp contexts initialize in both headed and headless modes.
func TestInitChromeDp(t *testing.T) {
	// Test with head
	Headless = false
	ctx, cancel := InitChromeDp()
	if ctx.Err() != nil {
		t.Errorf("Failed to initialize chromedp with head")
	}
	cancel()
	// Test headless
	Headless = true
	ctx, cancel = InitChromeDp()
	if ctx.Err() != nil {
		t.Errorf("Failed to initialize chromedp headlessly")
	}
	cancel()
}

// TestRefreshToken confirms coursebook tokens refresh under both headless settings.
func TestRefreshToken(t *testing.T) {
	if os.Getenv("LOGIN_NETID") == "" || os.Getenv("LOGIN_PASSWORD") == "" {
		t.Skip("LOGIN_NETID/LOGIN_PASSWORD not set; skipping RefreshToken integration test")
	}

	// Get a chromedp context
	ctx, cancel := InitChromeDp()
	defer cancel()
	// Try refreshing token
	headers := RefreshToken(ctx)
	// Make sure we successfully got a PTGSESSID cookie
	for _, cookie := range headers["Cookie"] {
		if strings.HasPrefix(cookie, "PTGSESSID") {
			return
		}
	}
	// Fail if no PTGSESSID cookie found
	t.Fatalf("Failed to get PTGSESSID cookie from RefreshToken!")
}
