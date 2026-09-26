#!/bin/sh

# for monthly tasks to run

# scrape, parse, and upload map locations
./api-tools -headless -verbose -scrape -map
./api-tools -headless -verbose -parse -map
./api-tools -headless -verbose -upload -map

# scrape, parse, and upload budgets
./api-tools -headless -verbose -scrape -budgets -useBackupBudgets
./api-tools -headless -verbose -parse -budgets -useBackupBudgets
./api-tools -headless -verbose -upload -budgets -useBackupBudgets
