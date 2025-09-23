#!/bin/sh

# for weekly tasks to run

# scrape, parse, and upload academic calendars
./api-tools -headless -verbose -scrape -academicCalendars
./api-tools -headless -verbose -parse -academicCalendars
./api-tools -headless -verbose -upload -academicCalendars
