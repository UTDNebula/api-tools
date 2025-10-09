#!/bin/sh

# for monthly tasks to run

# scrape, parse, and upload map locations
./api-tools -headless -verbose -scrape -mapFlag
./api-tools -headless -verbose -parse -mapFlag
./api-tools -headless -verbose -upload -mapFlag
