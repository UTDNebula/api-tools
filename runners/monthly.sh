#!/bin/sh

# for monthly tasks to run

# scrape, parse, and upload map locations
./api-tools -headless -verbose -scrape -map
./api-tools -headless -verbose -parse -map
./api-tools -headless -verbose -upload -map
