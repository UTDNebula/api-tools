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
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// Initializes Chrome DevTools Protocol
func InitChromeDp() (chromedpCtx context.Context, cancelFnc context.CancelFunc) {
	log.Printf("Initializing chromedp...")
	headlessEnv, present := os.LookupEnv("HEADLESS_MODE")
	doHeadless, _ := strconv.ParseBool(headlessEnv)
	if present && doHeadless {
		chromedpCtx, cancelFnc = chromedp.NewContext(context.Background())
		log.Printf("Initialized chromedp!")
	} else {
		allocCtx, _ := chromedp.NewExecAllocator(context.Background())
		chromedpCtx, cancelFnc = chromedp.NewContext(allocCtx)
	}
	return
}

// This function generates a fresh auth token and returns the new headers
func RefreshToken(chromedpCtx context.Context) map[string][]string {
	netID, present := os.LookupEnv("LOGIN_NETID")
	if !present {
		log.Panic("LOGIN_NETID is missing from .env!")
	}
	password, present := os.LookupEnv("LOGIN_PASSWORD")
	if !present {
		log.Panic("LOGIN_PASSWORD is missing from .env!")
	}

	VPrintf("Getting new token...")
	_, err := chromedp.RunResponse(chromedpCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			err := network.ClearBrowserCookies().Do(ctx)
			return err
		}),
		chromedp.Navigate(`https://wat.utdallas.edu/login`),
		chromedp.WaitVisible(`form#login-form`),
		chromedp.SendKeys(`input#netid`, netID),
		chromedp.SendKeys(`input#password`, password),
		chromedp.WaitVisible(`button#login-button`),
		chromedp.Click(`button#login-button`),
		chromedp.WaitVisible(`body`),
	)
	if err != nil {
		panic(err)
	}

	var cookieStrs []string
	_, err = chromedp.RunResponse(chromedpCtx,
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
	username, present := os.LookupEnv("LOGIN_ASTRA_USERNAME")
	if !present {
		log.Panic("LOGIN_ASTRA_USERNAME is missing from .env!")
	}
	password, present := os.LookupEnv("LOGIN_ASTRA_PASSWORD")
	if !present {
		log.Panic("LOGIN_ASTRA_PASSWORD is missing from .env!")
	}

	// Sign in
	VPrintf("Signing in...")
	_, err := chromedp.RunResponse(chromedpCtx,
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

	// Save all cookies to string
	cookieStr := ""
	_, err = chromedp.RunResponse(chromedpCtx,
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := network.GetCookies().Do(ctx)
			gotToken := false
			for _, cookie := range cookies {
				cookieStr = fmt.Sprintf("%s%s=%s; ", cookieStr, cookie.Name, cookie.Value)
				if cookie.Name == "UTXDallas.ASPXFORMSAUTH" {
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

// Attempts to run the given HTTP request with the given HTTP client, wrapping the request with a retry callback
func RetryHTTP(requestCreator func() *http.Request, client *http.Client, retryCallback func(res *http.Response, numRetries int)) (res *http.Response, err error) {
	// Retry loop for requests
	numRetries := 0
	for {
		// Perform HTTP request, retrying if we get a non-200 response code
		res, err = client.Do(requestCreator())
		// Retry handling
		if res.StatusCode != 200 {
			retryCallback(res, numRetries)
			numRetries++
			continue
		}
		break
	}
	return res, err
}
