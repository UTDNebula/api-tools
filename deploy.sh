#!/bin/sh

# service account
gcloud secrets versions access latest --secret="$SERVICE_ACCOUNT_SECRET_NAME" > service_account.json
gcloud auth activate-service-account --key-file=service_account.json
rm service_account.json

# .env
gcloud secrets versions access latest --secret="$ENV_SECRET_NAME" > .env

# Scrape, parse, and upload
./api-tools -scrape -mazevo -verbose
./api-tools -parse -mazevo -verbose
./api-tools -scrape -astra -verbose
./api-tools -parse -astra -verbose
./api-tools -upload -events -verbose
