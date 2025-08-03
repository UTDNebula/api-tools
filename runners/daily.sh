#!/bin/sh

# other daily activites
# ...

# Scrape, parse, and upload events
./api-tools -scrape -mazevo -verbose
./api-tools -parse -mazevo -verbose
./api-tools -scrape -astra -verbose
./api-tools -parse -astra -verbose
./api-tools -upload -events -verbose
