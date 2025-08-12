#!/bin/sh

# for daily tasks to run

# scrape, parse, and upload events
./api-tools -headless -verbose -scrape -mazevo
./api-tools -headless -verbose -parse -mazevo
./api-tools -headless -verbose -scrape -astra
./api-tools -headless -verbose -parse -astra
./api-tools -headless -verbose -upload -events

# generate and upload letters
./api-tools -headless -verbose -generate -letters
./api-tools -headless -verbose -upload -letters
