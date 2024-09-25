/*
	This file contains the code for the Astra scraper.
*/

package scrapers

import (
	"log"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/joho/godotenv"
)

func ScrapeAstra(outDir string) {

	// Load env vars
	if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
	}

	// Start chromedp
	chromedpCtx, cancel := utils.InitChromeDp()
	defer cancel()

	utils.SignInAstra(chromedpCtx)
}
