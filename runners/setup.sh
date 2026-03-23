#!/bin/sh

if [ "$ENVIRONMENT" = "gcp" ]; then
  # auth with service account
  gcloud secrets versions access latest --secret="$SERVICE_ACCOUNT_SECRET_NAME" > service_account.json
  gcloud auth activate-service-account --key-file=service_account.json
  rm service_account.json

  # use service account to access environment variables from GCP secrets, create .env
  gcloud secrets versions access latest --secret="$ENV_SECRET_NAME" > .env
else
  echo "ENVIRONMENT is set to '$ENVIRONMENT'. Skipping env setup."
fi

# run commands from the file path specified in the GCP run job's variable
sh "/app/runners/$RUNNER_SCRIPT_NAME"
