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

	// Flags for parsing
	parse := flag.Bool("parse", false, "Puts the tool into parsing mode.")
	csvDir := flag.String("csv", "./grade-data", "Alongside -parse, specifies the path to the directory of CSV files containing grade data.")
	skipValidation := flag.Bool("skipv", false, "Alongside -parse, signifies that the post-parsing validation should be skipped. Be careful with this!")

	// Flags for uploading data
	upload := flag.Bool("upload", false, "Puts the tool into upload mode.")
	replace := flag.Bool("replace", false, "Alongside -upload, specifies that uploaded data should replace existing data rather than being merged.")

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
	log.SetOutput(logFile)

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
		default:
			log.Panic("One of the -coursebook or -profiles flags must be set for scraping!")
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
