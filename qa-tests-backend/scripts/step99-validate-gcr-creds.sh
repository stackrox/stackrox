#!/bin/bash
set -eu
exit 1

pass qa-test-settings.properties


# Validate /gcp/stackrox-ci/sa/ci-gcr-scanner creds
PROJECT_ID="stackrox-ci"
SERVICE_ACCOUNT_NAME="ci-gcr-scanner"
SERVICE_ACCOUNT_ID="${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
CREDS_IDENT="qa-tests-backend/GOOGLE_CREDENTIALS_GCR_SCANNER.json"
KEY_FILE="/tmp/GOOGLE_CREDENTIALS_GCR_SCANNER.json"
pass $CREDS_IDENT > $KEY_FILE
jq -r . < $KEY_FILE
gcloud auth activate-service-account "$SERVICE_ACCOUNT_ID" --project="$PROJECT_ID" --key-file="$KEY_FILE"
gcloud auth list
docker logout us.gcr.io
docker pull us.gcr.io/stackrox-ci/qa/registry-image:0.3
gcloud auth configure-docker us.gcr.io
docker login us.gcr.io
docker image rm us.gcr.io/stackrox-ci/qa/registry-image:0.3
docker pull us.gcr.io/stackrox-ci/qa/registry-image:0.3


# Validate /gcp/stackrox-ci/sa/ci-gcr-no-access-test creds
PROJECT_ID="stackrox-ci"
SERVICE_ACCOUNT_NAME="ci-gcr-no-access-test"
SERVICE_ACCOUNT_ID="${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
CREDS_IDENT="qa-tests-backend/GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY.json"
KEY_FILE="/tmp/GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY.json"
pass $CREDS_IDENT > $KEY_FILE
jq -r . < $KEY_FILE
gcloud auth activate-service-account "$SERVICE_ACCOUNT_ID" --project="$PROJECT_ID" --key-file="$KEY_FILE"
gcloud auth list
docker logout us.gcr.io
docker pull us.gcr.io/stackrox-ci/qa/registry-image:0.3
gcloud auth configure-docker us.gcr.io
docker login us.gcr.io
docker image rm us.gcr.io/stackrox-ci/qa/registry-image:0.3
docker pull us.gcr.io/stackrox-ci/qa/registry-image:0.3
