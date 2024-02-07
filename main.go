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
)

func main() {

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

	// Flag for profile scraping
	scrapeProfiles := flag.Bool("profiles", false, "Alongside -scrape, signifies that professor profiles should be scraped.")
	// Flag for soc scraping
	scrapeOrganizations := flag.Bool("organizations", false, "Alongside -scrape, signifies that SOC organizations should be scraped.")
	// Flag for event scraping
	scrapeEvents := flag.Bool("events", false, "Alongside -scrape, signifies that events should be scraped.")
	// Flag for eval scraping
	scrapeEvals := flag.Bool("evaluations", false, "Alongside -scrape, signifies that course evaluations should be scraped. Requires coursebook to be scraped beforehand!")

	// Flags for parsing
	parse := flag.Bool("parse", false, "Puts the tool into parsing mode.")
	csvDir := flag.String("csv", "./grade-data", "Alongside -parse, specifies the path to the directory of CSV files containing grade data.")
	skipValidation := flag.Bool("skipv", false, "Alongside -parse, signifies that the post-parsing validation should be skipped. Be careful with this!")

	// Flags for uploading data
	upload := flag.Bool("upload", false, "Puts the tool into upload mode.")
	replace := flag.Bool("replace", false, "Alongside -upload, specifies that uploaded data should replace existing data rather than being merged.")

	// Flags for logging
	verbose := flag.Bool("verbose", false, "Enables verbose logging, good for debugging purposes.")

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
	switch {
	case *scrape:
		switch {
		case *scrapeProfiles:
			scrapers.ScrapeProfiles(*outDir)
		case *scrapeCoursebook:
			if *term == "" {
				log.Panic("No term specified for coursebook scraping! Use -term to specify.")
			}
			scrapers.ScrapeCoursebook(*term, *startPrefix, *outDir)
		case *scrapeOrganizations:
			scrapers.ScrapeOrganizations(*outDir)
		case *scrapeEvents:
			scrapers.ScrapeEvents(*outDir)
		case *scrapeEvals:
			scrapers.ScrapeEvals(*inDir)
		default:
			log.Panic("You must specify which type of scraping you would like to perform with one of the scraping flags!")
		}
	case *parse:
		parser.Parse(*inDir, *outDir, *csvDir, *skipValidation)
	case *upload:
		uploader.Upload(*inDir, *replace)
	default:
		flag.PrintDefaults()
		return
	}
}
