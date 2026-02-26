#!/bin/sh

# for weekly tasks to run

# scrape, parse, and upload academic calendars
./api-tools -headless -verbose -scrape -academicCalendars
./api-tools -headless -verbose -parse -academicCalendars
./api-tools -headless -verbose -upload -academicCalendars

# scrape, parse, and upload discount programs
./api-tools -headless -verbose -scrape -discounts
./api-tools -headless -verbose -parse -discounts
./api-tools -headless -verbose -upload -discounts
