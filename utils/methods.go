/*
	This file contains utility methods used throughout various files in this repo.
*/

package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

var Headless = true

// Finds .env value and produces proper error if not found
func GetEnv(name string) (string, error) {
	value, exists := os.LookupEnv(name)
	if !exists || value == "" {
		return "", errors.New(name + " is missing from .env!")
	}
	return value, nil
}

// Initializes Chrome DevTools Protocol
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

// This function generates a fresh auth token and returns the new headers
func RefreshToken(chromedpCtx context.Context) map[string][]string {
	netID, err := GetEnv("LOGIN_NETID")
	if err != nil {
		panic(err)
	}
	password, err := GetEnv("LOGIN_PASSWORD")
	if err != nil {
		panic(err)
	}

	delayedRetryCallback := func(numRetries int) {
		time.Sleep(250 * time.Millisecond * time.Duration(numRetries))
	}

	VPrintf("Getting new token...")
	err = Retry(func() error {
		r, err := chromedp.RunResponse(chromedpCtx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				err := network.ClearBrowserCookies().Do(ctx)
				return err
			}),
			chromedp.Navigate(`https://wat.utdallas.edu/login`),
			chromedp.SendKeys(`input#netid`, netID),
			chromedp.SendKeys(`input#password`, password),
			chromedp.Click(`button#login-button`),
		)
		if r != nil && r.Status != 200 {
			return errors.New("Non-200 response status code")
		}
		return err
	}, 3, delayedRetryCallback)

	if err != nil {
		panic(err)
	}

	time.Sleep(250 * time.Millisecond)

	var cookieStrs []string

	err = Retry(func() error {
		r, err := chromedp.RunResponse(chromedpCtx,
			chromedp.Navigate(`https://coursebook.utdallas.edu/`),
			chromedp.ActionFunc(func(ctx context.Context) error {
				cookies, err := network.GetCookies().Do(ctx)
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
				return err
			}),
		)
		if r != nil && r.Status != 200 {
			return errors.New("Non-200 response status code")
		}
		return err
	}, 3, delayedRetryCallback)

	if err != nil {
		panic(err)
	}

	return map[string][]string{
		"Host":            {"coursebook.utdallas.edu"},
		"User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/110.0"},
		"Accept":          {"text/html"},
		"Accept-Language": {"en-US"},
		"Content-Type":    {"application/x-www-form-urlencoded"},
		"Cookie":          cookieStrs,
		"Connection":      {"keep-alive"},
	}
}

// This function signs into Astra
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

// Encodes and writes the given data as tab-indented JSON to the given filepath.
func WriteJSON(filepath string, data interface{}) error {
	fptr, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer fptr.Close()
	encoder := json.NewEncoder(fptr)
	encoder.SetIndent("", "\t")
	encoder.Encode(data)
	return nil
}

// Recursively gets the filepath of every file with the given extension, using the given directory as the root.
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

// Removes standard whitespace characters (space, tab, newline, carriage return) from a given string.
func TrimWhitespace(text string) string {
	return strings.Trim(text, " \t\n\r")
}

// Gets all of the values from a given map.
func GetMapValues[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

// Gets all of the keys from a given map.
func GetMapKeys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

// Creates a regexp with MustCompile() using a sprintf input.
func Regexpf(format string, vars ...interface{}) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(format, vars...))
}

// Attempts to retry running the given error-returning function up to a maximum number of retries, at which point the last error is returned. A callback is called between each retry.
func Retry(action func() error, maxRetries int, retryCallback func(numRetries int)) error {
	for retries := 1; ; retries++ {
		// Perform the action
		err := action()
		if err == nil || retries > maxRetries {
			return err
		}
		retryCallback(retries)
	}
}

// Get all the available course prefixes
func GetCoursePrefixes(chromedpCtx context.Context) []string {
	// Refresh the token
	// refreshToken(chromedpCtx)

	log.Printf("Finding course prefix nodes...")

	var coursePrefixes []string
	var coursePrefixNodes []*cdp.Node

	// Get option elements for course prefix dropdown
	err := chromedp.Run(chromedpCtx,
		chromedp.Navigate("https://coursebook.utdallas.edu"),
		chromedp.Nodes("select#combobox_cp option", &coursePrefixNodes, chromedp.ByQueryAll),
	)

	if err != nil {
		log.Panic(err)
	}

	log.Println("Found the course prefix nodes!")

	log.Println("Finding course prefixes...")

	// Remove the first option due to it being empty
	coursePrefixNodes = coursePrefixNodes[1:]

	// Get the value of each option and append to coursePrefixes
	for _, node := range coursePrefixNodes {
		coursePrefixes = append(coursePrefixes, node.AttributeValue("value"))
	}

	log.Println("Found the course prefixes!")

	return coursePrefixes
}

func ConvertFromInterface[T string | float64](value interface{}) *T {
	if parsed, ok := value.(T); ok {
		return &parsed
	}
	return nil
}
