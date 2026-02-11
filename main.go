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
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	godotenv.Load()

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
	// Flag for discount programs scraping
	scrapeDiscounts := flag.Bool("discounts", false, "Alongside -scrape, signifies that discount programs should be scraped.")
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
	degrees := flag.Bool("degrees", false, "Alongside -scrape or -parse, signifies that the degrees should be scraped/parsed.")

	// Flags for parsing
	parse := flag.Bool("parse", false, "Puts the tool into parsing mode.")
	csvDir := flag.String("csv", "./grade-data", "Alongside -parse, specifies the path to the directory of CSV files containing grade data.")
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
	logFile, err := os.Create(fmt.Sprintf("./logs/%d-%d-%dT%d-%d-%d.log", month, day, year, hour, min, sec))

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
		default:
			parser.Parse(*inDir, *outDir, *csvDir, *skipValidation)
		}
	case *upload:
		switch {
		case *events:
			uploader.UploadEvents(*inDir)
		case *mapFlag:
			uploader.UploadMapLocations(*inDir)
		case *academicCalendars:
			uploader.UploadAcademicCalendars(*inDir)
		default:
			uploader.Upload(*inDir, *replace, *staticOnly)
		}
	default:
		flag.PrintDefaults()
		return
	}
}
