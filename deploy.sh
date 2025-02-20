#!/bin/sh

# .env
API_TOOLS_ENV=$(gcloud secrets versions access latest --secret="$ENV_SECRET_NAME")
echo "$API_TOOLS_ENV" > .env

# Scrape, parse, and upload
./api-tools -scrape -mazevo -verbose
#./api-tools -parse -mazevo -verbose
./api-tools -scrape -astra -verbose
#./api-tools -parse -astra -verbose
#./api-tools -upload -events -verbose
