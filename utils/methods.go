// Package utils provides shared helpers for scraping, parsing, and uploading workflows.
package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const (
	coursebookMaxRetries = 8
)

// Headless toggles whether chromedp runs without a visible browser window.
var Headless = true

// GetEnv finds an environment variable and returns an error when it is unset.
func GetEnv(name string) (string, error) {
	value, exists := os.LookupEnv(name)
	if !exists || value == "" {
		return "", errors.New(name + " is missing from .env!")
	}
	return value, nil
}

// InitChromeDp configures and returns a chromedp context with optional headless settings.
func InitChromeDp() (chromedpCtx context.Context, cancelFnc context.CancelFunc) {
	log.Printf("Initializing chromedp...")
	if Headless {
		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-gpu", true),
		)
		allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
		chromedpCtx, cancelFnc = chromedp.NewContext(allocCtx)
	} else {
		allocCtx, _ := chromedp.NewExecAllocator(context.Background())
		chromedpCtx, cancelFnc = chromedp.NewContext(allocCtx)
	}
	log.Printf("Initialized chromedp!")
	return chromedpCtx, cancelFnc
}

// RefreshToken logs into CourseBook and returns headers containing a fresh session token.
func RefreshToken(chromedpCtx context.Context) (map[string][]string, error) {
	netID, err := GetEnv("LOGIN_NETID")
	if err != nil {
		return nil, err
	}
	password, err := GetEnv("LOGIN_PASSWORD")
	if err != nil {
		return nil, err
	}

	VPrintf("Getting new token...")

	// Retry login loop with exponential backoff
	for attempt := 0; attempt <= coursebookMaxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			VPrintf("Refresh token error: %v, backing off for %v", err, wait)
			VPrintf("[Refresh Token Retry] Attempt %d of %d for Login", attempt, coursebookMaxRetries)
			time.Sleep(wait)
		}

		start := time.Now()
		r, loginErr := chromedp.RunResponse(chromedpCtx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return network.ClearBrowserCookies().Do(ctx)
			}),
			chromedp.Navigate(`https://wat.utdallas.edu/login`),
			chromedp.SendKeys(`input#netid`, netID),
			chromedp.SendKeys(`input#password`, password),
			chromedp.Click(`button#login-button`),
		)
		dur := time.Since(start)

		if r != nil {
			if r.Status != 200 {
				err = fmt.Errorf("non-200 response status code: got code %d", r.Status)
			} else {
				VPrintf("[Refresh Token Success] Refresh token login took %v", dur)
				err = nil
			}
		} else if loginErr != nil {
			err = loginErr
			VPrintf("[Refresh Token Error] Refresh token login failed: %v", err)
		}

		// Success, break
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("refresh token login failed after %d attempts: %w", coursebookMaxRetries+1, err)
	}

	time.Sleep(250 * time.Millisecond) // TODO: It might be more robust to not wait a fixed amount of time here

	var cookieStrs []string

	// Get cookie loop with exponential backoff
	for attempt := 0; attempt <= coursebookMaxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			VPrintf("Refresh token error: %v, backing off for %v", err, wait)
			VPrintf("[Refresh Token Retry] Attempt %d of %d for Cookie Retrieval", attempt, coursebookMaxRetries)
			time.Sleep(wait)
		}

		start := time.Now()
		r, cookieErr := chromedp.RunResponse(chromedpCtx,
			chromedp.Navigate(`https://coursebook.utdallas.edu/`),
			chromedp.ActionFunc(func(ctx context.Context) error {
				cookies, actionErr := network.GetCookies().Do(ctx)
				if actionErr != nil {
					return actionErr
				}
				cookieStrs = make([]string, len(cookies))
				gotToken := false
				for i, cookie := range cookies {
					cookieStrs[i] = fmt.Sprintf("%s=%s", cookie.Name, cookie.Value)
					if cookie.Name == "PTGSESSID" {
						VPrintf("Got new token: PTGSESSID = %s", cookie.Value)
						gotToken = true
					}
				}
				if !gotToken {
					return errors.New("failed to get a new token")
				}
				return nil
			}),
		)
		dur := time.Since(start)

		if r != nil {
			if r.Status != 200 {
				err = fmt.Errorf("non-200 response status code: got code %d", r.Status)
			} else {
				VPrintf("[Refresh Token Success] Refresh token cookie retrieval took %v", dur)
				err = nil
			}
		} else if cookieErr != nil {
			err = cookieErr
			VPrintf("[Refresh Token Error] Refresh token cookie retrieval failed: %v", err)
		}

		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("refresh token cookie retrieval failed after %d attempts: %w", coursebookMaxRetries+1, err)
	}

	return map[string][]string{
		"Host":            {"coursebook.utdallas.edu"},
		"User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/110.0"},
		"Accept":          {"text/html"},
		"Accept-Language": {"en-US"},
		"Content-Type":    {"application/x-www-form-urlencoded"},
		"Cookie":          cookieStrs,
		"Connection":      {"keep-alive"},
	}, nil
}

// RefreshAstraToken signs into Astra and returns headers containing authentication cookies.
func RefreshAstraToken(chromedpCtx context.Context) map[string][]string {
	// Get username and password
	username, err := GetEnv("LOGIN_ASTRA_USERNAME")
	if err != nil {
		log.Panic("LOGIN_ASTRA_USERNAME is missing from .env!")
	}
	password, err := GetEnv("LOGIN_ASTRA_PASSWORD")
	if err != nil {
		log.Panic("LOGIN_ASTRA_PASSWORD is missing from .env!")
	}

	// Sign in
	VPrintf("Signing in...")
	_, err = chromedp.RunResponse(chromedpCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			err := network.ClearBrowserCookies().Do(ctx)
			return err
		}),
		chromedp.Navigate(`https://www.aaiscloud.com/UTXDallas/logon.aspx?ReturnUrl=%2futxdallas%2fcalendars%2fdailygridcalendar.aspx`),
		chromedp.WaitVisible(`input#userNameField-inputEl`),
		chromedp.SendKeys(`input#userNameField-inputEl`, username),
		chromedp.SendKeys(`input#textfield-1029-inputEl`, password),
		chromedp.WaitVisible(`a#logonButton`),
		chromedp.Click(`a#logonButton`),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
	)
	if err != nil {
		panic(err)
	}

	time.Sleep(250 * time.Millisecond)

	// Save all cookies to string
	cookieStr := ""
	err = chromedp.Run(chromedpCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := network.GetCookies().Do(ctx)
			if err != nil {
				return err
			}
			gotToken := false
			for _, cookie := range cookies {
				cookieStr = fmt.Sprintf("%s%s=%s; ", cookieStr, cookie.Name, cookie.Value)
				if cookie.Name == "UTXDallas.ASPXFORMSAUTH" {
					VPrintf("Got new token: UTXDallas.ASPXFORMSAUTH = %s", cookie.Value)
					gotToken = true
				}
			}
			if !gotToken {
				return errors.New("failed to get a new token")
			}
			return nil
		}),
	)
	if err != nil {
		panic(err)
	}

	// Return headers, copied from a request the actual site made
	return map[string][]string{
		"Host":                      {"www.aaiscloud.com"},
		"User-Agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/110.0"},
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/png,image/svg+xml,*/*;q=0.8"},
		"Accept-Language":           {"en-US,en;q=0.5"},
		"Accept-Encoding":           {"gzip, deflate, br, zstd"},
		"Connection":                {"keep-alive"},
		"Cookie":                    {cookieStr},
		"Upgrade-Insecure-Requests": {"1"},
		"Sec-Fetch-Dest":            {"document"},
		"Sec-Fetch-Mode":            {"navigate"},
		"Sec-Fetch-Site":            {"none"},
		"Sec-Fetch-User":            {"?1"},
		"Priority":                  {"u=0, i"},
	}
}

// WriteJSON encodes data as indented JSON and writes it to filepath.
func WriteJSON(filepath string, data interface{}) error {
	fptr, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer fptr.Close()
	encoder := json.NewEncoder(fptr)
	encoder.SetIndent("", "\t")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(data)
}

// GetAllFilesWithExtension recursively gathers file paths within inDir that match extension.
func GetAllFilesWithExtension(inDir string, extension string) []string {
	var filePaths []string
	err := filepath.WalkDir(inDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Add any html files (excluding evals) to sectionFilePaths
		if filepath.Ext(path) == extension {
			filePaths = append(filePaths, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return filePaths
}

// TrimWhitespace removes spaces, tabs, newlines, and carriage returns from the provided string.
func TrimWhitespace(text string) string {
	return strings.Trim(text, " \t\n\r")
}

// GetMapValues returns a slice of all map values.
func GetMapValues[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

// GetMapKeys returns a slice of all map keys.
func GetMapKeys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

// Regexpf formats and compiles a regular expression pattern using fmt.Sprintf semantics.
func Regexpf(format string, vars ...interface{}) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(format, vars...))
}

// GetCoursePrefixes retrieves all course prefix values from CourseBook.
func GetCoursePrefixes(chromedpCtx context.Context) ([]string, error) {
	// Might need to refresh the token every time we get new course prefixes in the future
	// refreshToken(chromedpCtx)

	var coursePrefixes []string
	log.Println("Finding course prefixes...")

	var err error
	maxRetries := 8

	for attempt := 1; attempt <= maxRetries; attempt++ {
		coursePrefixes = nil // Reset course prefixes for each attempt

		// Get option elements for course prefix dropdown
		_, err = chromedp.RunResponse(chromedpCtx,
			chromedp.Navigate("https://coursebook.utdallas.edu"),
			chromedp.QueryAfter("select#combobox_cp option",
				func(ctx context.Context, _ runtime.ExecutionContextID, nodes ...*cdp.Node) error {
					for _, node := range nodes[1:] {
						coursePrefixes = append(coursePrefixes, node.AttributeValue("value"))
					}
					return nil
				},
			),
		)

		if err == nil {
			break // Success, exit retry loop
		}

		VPrintf("[Get Course Prefixes Failed] %v", err)

		// Exponential backoff
		wait := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		VPrintf("Coursebook load error, waiting %v (attempt %d of %d)", wait, attempt, maxRetries)
		time.Sleep(wait)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch course prefixes after %d attempts: %w", maxRetries, err)
	}
	log.Printf("Found the %d course prefixes!", len(coursePrefixes))
	return coursePrefixes, nil
}

// ConvertFromInterface attempts to convert a value into the requested type and returns a pointer when successful.
func ConvertFromInterface[T string | float64](value any) *T {
	if parsed, ok := value.(T); ok {
		return &parsed
	}
	return nil
}
