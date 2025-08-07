#!/bin/sh

# for daily tasks to run

# scrape, parse, and upload events
./api-tools -headless -verbose -scrape -mazevo -verbose
./api-tools -headless -verbose -parse -mazevo -verbose
./api-tools -headless -verbose -scrape -astra -verbose
./api-tools -headless -verbose -parse -astra -verbose
./api-tools -headless -verbose -upload -events -verbose
