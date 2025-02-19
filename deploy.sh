#!/bin/sh

# .env
API_TOOLS_ENV=$(gcloud secrets versions access latest --secret="$ENV_SECRET_NAME")
echo "$API_TOOLS_ENV" > .env

# Scrape, parse, and upload
#exec ./api-tools -scrape -mazevo -verbose
#exec ./api-tools -parse -mazevo -verbose
exec ./api-tools -scrape -astra -verbose
#exec ./api-tools -parse -astra -verbose
#exec ./api-tools -upload -events -verbose
