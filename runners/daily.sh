#!/bin/sh

# for daily tasks to run

# scrape, parse, and upload events
./api-tools -scrape -mazevo -verbose
./api-tools -parse -mazevo -verbose
./api-tools -scrape -astra -verbose
./api-tools -parse -astra -verbose
./api-tools -upload -events -verbose
