package scrapers

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/UTDNebula/api-tools/schema"
	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	socBaseUrl         = `https://cometmail.sharepoint.com`
	socLoginUrl        = socBaseUrl + `/sites/StudentOrganizationCenterSP/Lists/Student%20Organization%20Directory/All%20Items%20gallery.aspx`
	localPartCharClass = `[:alnum:]!#$%&'*+/=?^_` + "`" + `{|}~-`
	subdomainPattern   = `([[:alnum:]]([[:alnum:]-]*[[:alnum:]])?\.)+`
	topdomainPattern   = `[[:alnum:]]([[:alnum:]-]*[[:alnum:]])?`
)

var (
	baseUrlStruct, _ = url.Parse(socBaseUrl)
	localPartPattern = fmt.Sprintf(`[%[1]s]+(\.[%[1]s]+)*`, localPartCharClass)
	emailRegex       = regexp.MustCompile(fmt.Sprintf(`%s@%s%s`, localPartPattern, subdomainPattern, topdomainPattern))
)

func ScrapeOrganizations(outdir string) {
	log.Println("Scraping SOC ...")
	if err := godotenv.Load(); err != nil {
		panic(errors.New("error loading .env file"))
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false))
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	// ensure cleanup occurs
	defer cancel()

	if err := loginToSoc(ctx); err != nil {
		panic(err)
	}
	if err := scrapeData(ctx, outdir); err != nil {
		panic(err)
	}
}

func lookupEnvWithError(name string) (string, error) {
	value, exists := os.LookupEnv(name)
	if !exists {
		return "", errors.New(name + " is missing from .env!")
	}
	return value, nil
}

func loginToSoc(ctx context.Context) error {
	log.Println("Logging into SOC ...")
	netID, err := lookupEnvWithError("LOGIN_NETID")
	if err != nil {
		return err
	}
	password, err := lookupEnvWithError("LOGIN_PASSWORD")
	if err != nil {
		return err
	}

	return chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return network.ClearBrowserCookies().Do(ctx)
		}),
		chromedp.Navigate(socLoginUrl),
		chromedp.SendKeys(`input[type="email"]`, netID+"@utdallas.edu"),
		chromedp.Click(`input[type="submit"]`),
		chromedp.SendKeys(`input[type="password"]`, password),
		// wait for sign in button to load (regular WaitVisible and WaitReady methods do not work)
		chromedp.Sleep(1*time.Second),
		chromedp.Click(`input[type="submit"]`),
		chromedp.Sleep(1*time.Second),
		chromedp.Click("button.auth-button"),
		chromedp.WaitReady(`body`),
	)
}

func scrapeData(ctx context.Context, outdir string) error {
	log.Println("Scraping data ...")
	// download file method adapted from https://github.com/chromedp/examples/blob/master/download_file/main.go
	timedCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	done := make(chan string, 1)
	// listen for download events
	chromedp.ListenTarget(timedCtx, func(v interface{}) {
		ev, ok := v.(*browser.EventDownloadProgress)
		if !ok {
			return
		}
		if ev.State == browser.DownloadProgressStateCompleted {
			// stop listening for further download events and send guid
			cancel()
			done <- ev.GUID
			close(done)
		}
	})

	tempDir := filepath.Join(outdir, "tmp")
	if err := chromedp.Run(ctx,
		chromedp.Sleep(1*time.Second),
		chromedp.Click(`button[name="Export"]`, chromedp.NodeReady),
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).WithDownloadPath(tempDir).WithEventsEnabled(true),
		chromedp.Sleep(1*time.Second),
		chromedp.Click(`button[name="Export to CSV"]`, chromedp.NodeReady),
	); err != nil {
		return err
	}

	// get GUID of download and reconstruct path
	guid, _ := <-done
	guidPath := filepath.Join(tempDir, guid)
	defer func() {
		// remove temp file and directory
		os.Remove(guidPath)
	}()

	outPath := filepath.Join(outdir, "organizations.jsonl")

	if err := processCsv(ctx, guidPath, outPath); err != nil {
		return err
	}

	return nil
}

func processCsv(ctx context.Context, inputPath string, storageFilePath string) error {
	// open csv for reading
	csvFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}

	// init csv reader
	bufReader := bufio.NewReader(csvFile)
	// discard headers
	if _, _, err := bufReader.ReadLine(); err != nil {
		return err
	}
	csvReader := csv.NewReader(bufReader)

	// write to json
	storageFile, err := os.Create(storageFilePath)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(bufio.NewWriter(storageFile))

	var _ []*schema.Organization
	// process each row of csv
	for i := 1; true; i++ {
		entry, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		log.Printf("Processing row %d\n", i)
		org, err := parseCsvRecord(ctx, entry)
		if err != nil {
			return err
		}
		if err := encoder.Encode(org); err != nil {
			return err
		}
	}

	if err := csvFile.Close(); err != nil {
		return err
	}

	if err := storageFile.Close(); err != nil {
		return err
	}

	return nil
}

func parseCsvRecord(ctx context.Context, entry []string) (*schema.Organization, error) {
	// initial cleaning
	for i, v := range entry {
		v = strings.ReplaceAll(v, "\u0026", "")
		v = strings.TrimSpace(v)
		entry[i] = v
	}

	imageData, err := retrieveImage(ctx, entry[5])
	if err != nil {
		log.Printf("Error retrieving image for %s: %v\n", entry[0], err)
	}
	return &schema.Organization{
		Id:             schema.IdWrapper{Id: primitive.NewObjectID()},
		Title:          entry[0],
		Categories:     parseCategories(entry[1]),
		Description:    entry[2],
		President_name: entry[3],
		Emails:         parseEmails(entry[4]),
		Picture_data:   imageData,
	}, nil
}

func parseCategories(cats string) []string {
	cats = strings.TrimLeft(cats, "[")
	cats = strings.TrimRight(cats, "]")
	// strange character appears in csv; need to remove it
	cats = strings.ReplaceAll(cats, `"`, "")
	// split by comma
	catsArray := strings.Split(cats, ",")
	// strip whitespace from ends
	for j, v := range catsArray {
		catsArray[j] = strings.TrimSpace(v)
	}

	return catsArray
}

func parseEmails(emails string) []string {
	return emailRegex.FindAllString(emails, -1)
}

func retrieveImage(ctx context.Context, imageUri string) (string, error) {
	if imageUri == "" {
		return "", nil
	}

	urlStruct, err := url.Parse(imageUri)
	if err != nil {
		return "", err
	}

	requestUrl := baseUrlStruct.ResolveReference(urlStruct).String()

	//log.Printf("loading image %s\n", requestUrl)
	// method adapted from https://github.com/chromedp/examples/blob/master/download_image/main.go

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	done := make(chan bool)

	// this will be used to capture the request id for matching network events
	var requestID network.RequestID

	// listen for network requests and choose desired
	chromedp.ListenTarget(ctx, func(v interface{}) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			if ev.Request.URL == requestUrl {
				requestID = ev.RequestID
			}
		case *network.EventLoadingFinished:
			if ev.RequestID == requestID {
				close(done)
			}
		}
	})

	if err := chromedp.Run(ctx, chromedp.Navigate(requestUrl)); err != nil {
		log.Printf("Error navigating to %s: %v\n", requestUrl, err)
		return "", err
	}

	// wait for image request to finish
	<-done
	//log.Printf("Done retrieving image from %s\n", requestUrl)

	var buf []byte
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		buf, err = network.GetResponseBody(requestID).Do(ctx)
		if err != nil {
			log.Printf("Error getting response body for %s: %v\n", requestUrl, err)
		}
		return err
	})); err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(buf)
	// get response body
	return encoded, nil
}
