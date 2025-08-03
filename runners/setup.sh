#!/bin/sh

# service account
gcloud secrets versions access latest --secret="$SERVICE_ACCOUNT_SECRET_NAME" > service_account.json
gcloud auth activate-service-account --key-file=service_account.json
rm service_account.json

# .env
gcloud secrets versions access latest --secret="$ENV_SECRET_NAME" > .env

# run commands from file specified in GCP
sh "/app/runners/$RUNNER_SCRIPT_NAME"
