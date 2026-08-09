package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/UTDNebula/api-tools/parser"
	"github.com/UTDNebula/api-tools/scrapers"
	"github.com/UTDNebula/api-tools/uploader"
	"github.com/UTDNebula/api-tools/utils"
	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	godotenv.Load()

	// Set up Sentry
	sentryDsn, err := utils.GetEnv("SENTRY_DSN")
	if err != nil {
		sentryDsn = ""
	}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              sentryDsn,
		TracesSampleRate: 1.0,
		EnableTracing:    true,
	}); err != nil {
		log.Printf("Sentry initialization failed: %v\n", err)
	}
	defer sentry.Flush(2 * time.Second)

	// Setup flags

	// I/O Flags
	inDir := flag.String("i", "./data", "The directory to read data from. Defaults to ./data.")
	outDir := flag.String("o", "./data", "The directory to write resulting data to. Defaults to ./data.")
	logDir := flag.String("l", "./logs", "The directory to write logs to. Defaults to ./logs.")

	// Flags for all scraping
	scrape := flag.Bool("scrape", false, "Puts the tool into scraping mode.")

	// Flags for coursebook scraping
	scrapeCoursebook := flag.Bool("coursebook", false, "Alongside -scrape, signifies that coursebook should be scraped.")
	term := flag.String("term", "", "Alongside -coursebook, specifies the term to scrape, i.e. 23S")
	startPrefix := flag.String("startprefix", "", "Alongside -coursebook, specifies the course prefix to start scraping from, i.e. cp_span")
	resume := flag.Bool("resume", false, "Alongside -coursebook, signifies that scraping should begin at the last complete prefix and should not re-scrape existing data")

	// Flag for profile scraping
	scrapeProfiles := flag.Bool("profiles", false, "Alongside -scrape, signifies that professor profiles should be scraped.")
	// Flag for discount programs scraping, parsing, and uploading
	scrapeDiscounts := flag.Bool("discounts", false, "Alongside -scrape, -parse, or -upload, signifies that discount programs should be scraped/parsed/uploaded.")
	// Flag for calendar scraping and parsing
	cometCalendar := flag.Bool("cometCalendar", false, "Alongside -scrape or -parse, signifies that the Comet Calendar should be scraped/parsed.")
	// Flag for astra scraping and parsing
	astra := flag.Bool("astra", false, "Alongside -scrape or -parse, signifies that Astra should be scraped/parsed.")
	// Flag for mazevo scraping and parsing
	mazevo := flag.Bool("mazevo", false, "Alongside -scrape or -parse, signifies that Mazevo should be scraped/parsed.")
	// Flag for map scraping, parsing, and uploading
	mapFlag := flag.Bool("map", false, "Alongside -scrape, -parse, or -upload, signifies that the UTD map should be scraped/parsed/uploaded.")
	// Flag for academic calendar scraping
	academicCalendars := flag.Bool("academicCalendars", false, "Alongside -scrape, -parse, or -upload, signifies that the academic calendars should be scraped/parsed/uploaded.")
	// Flag for degree scraping and parsing
	degrees := flag.Bool("degrees", false, "Alongside -scrape, -parse, or -upload. Signifies that the degrees should be scraped/parsed/uploaded.")
	// Flag for budget scraping
	budgets := flag.Bool("budgets", false, "Alongside -scrape, -parse, or -upload, signifies that the budgets should be scraped/parsed/uploaded.")

	// Flags for parsing
	parse := flag.Bool("parse", false, "Puts the tool into parsing mode.")
	gradesDir := flag.String("gradesDir", "./static-data/grades", "Alongside -parse, specifies the path to the directory of CSV files containing grade data.")
	useBackupBudgets := flag.Bool("useBackupBudgets", false, "Alongside -parse, specifies that backup budget data should also be parsed.")
	budgetsDir := flag.String("budgetsDir", "./static-data/budgets", "Alongside -parse, specifies the path to the directory of PDF files containing budget data.")
	skipValidation := flag.Bool("skipv", false, "Alongside -parse, signifies that the post-parsing validation should be skipped. Be careful with this!")

	// Flags for uploading data
	upload := flag.Bool("upload", false, "Puts the tool into upload mode.")
	replace := flag.Bool("replace", false, "Alongside -upload, specifies that uploaded data should replace existing data rather than being merged.")
	staticOnly := flag.Bool("static", false, "Alongside -upload, specifies that we should only build and upload the static aggregations.")
	events := flag.Bool("events", false, "Alongside -upload, signifies that Astra, Mazevo, and the Comet Calendar should be uploaded.")

	// Flags for logging
	verbose := flag.Bool("verbose", false, "Enables verbose logging, good for debugging purposes.")

	// Flag for headless mode
	headless := flag.Bool("headless", false, "Enables headless mode for chromedp.")

	// Parse flags
	flag.Parse()

	// Make log dir if it doesn't already exist
	if _, err := os.Stat(*logDir); err != nil {
		os.Mkdir(*logDir, os.ModePerm)
	}

	// Make new log file for this session using timestamp
	dateTime := time.Now()
	year, month, day := dateTime.Date()
	hour, min, sec := dateTime.Clock()
	logFile, err := os.Create(fmt.Sprintf("./logs/%d-%02d-%02dT%02d-%02d-%02d.log", year, month, day, hour, min, sec))

	if err != nil {
		log.Fatal(err)
	}

	defer logFile.Close()
	// Set logging output destination to a SplitWriter that writes to both the log file and stdout
	log.SetOutput(utils.NewSplitWriter(logFile, os.Stdout))
	// Do verbose logging if verbose flag specified
	if *verbose {
		log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile | utils.Lverbose)
	} else {
		log.SetFlags(log.Ltime)
	}

	// Perform actions based on flags
	utils.Headless = *headless
	switch {
	case *scrape:
		switch {
		case *scrapeProfiles:
			scrapers.ScrapeProfiles(*outDir)
		case *scrapeCoursebook:
			if *term == "" {
				log.Panic("No term specified for coursebook scraping! Use -term to specify.")
			}
			scrapers.ScrapeCoursebook(*term, *startPrefix, *outDir, *resume)
		case *scrapeDiscounts:
			scrapers.ScrapeDiscounts(*outDir)
		case *cometCalendar:
			scrapers.ScrapeCometCalendar(*outDir)
		case *astra:
			scrapers.ScrapeAstra(*outDir)
		case *mazevo:
			scrapers.ScrapeMazevo(*outDir)
		case *mapFlag:
			scrapers.ScrapeMapLocations(*outDir)
		case *academicCalendars:
			scrapers.ScrapeAcademicCalendars(*outDir)
		case *degrees:
			scrapers.ScrapeDegrees(*outDir)
		case *budgets:
			scrapers.ScrapeBudgets(*outDir)
		default:
			log.Panic("You must specify which type of scraping you would like to perform with one of the scraping flags!")
		}
	case *parse:
		switch {
		case *cometCalendar:
			parser.ParseCometCalendar(*inDir, *outDir)
		case *astra:
			parser.ParseAstra(*inDir, *outDir)
		case *mazevo:
			parser.ParseMazevo(*inDir, *outDir)
		case *mapFlag:
			parser.ParseMapLocations(*inDir, *outDir)
		case *academicCalendars:
			parser.ParseAcademicCalendars(*inDir, *outDir)
		case *scrapeDiscounts:
			parser.ParseDiscounts(*inDir, *outDir)
		case *degrees:
			parser.ParseDegrees(*inDir, *outDir)
		case *budgets:
			parser.ParseBudgets(*inDir, *outDir, *budgetsDir, *useBackupBudgets)
		default:
			parser.Parse(*inDir, *outDir, *gradesDir, *skipValidation)
		}
	case *upload:
		switch {
		case *events:
			uploader.UploadEvents(*inDir)
		case *mapFlag:
			uploader.UploadMapLocations(*inDir)
		case *academicCalendars:
			uploader.UploadAcademicCalendars(*inDir)
		case *scrapeDiscounts:
			uploader.UploadDiscounts(*inDir)
		case *degrees:
			uploader.UploadDegrees(*inDir)
		case *budgets:
			uploader.UploadBudgets(*inDir)
		default:
			uploader.Upload(*inDir, *replace, *staticOnly)
		}
	default:
		flag.PrintDefaults()
		return
	}
}
