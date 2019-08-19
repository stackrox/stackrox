#!/usr/bin/env bash

set -ex

if [[ -z $DB_URL ]]; then
    echo "DB_URL must be defined to download recorder backup"
    exit 1
fi

if [[ -z $GOOGLE_APPLICATION_CREDENTIALS ]]; then
    echo "$GOOGLE_APPLICATION_CREDENTIALS must be defined to download recorder backup"
    exit 1
fi

./google-cloud-sdk/bin/gcloud auth activate-service-account --key-file $GOOGLE_APPLICATION_CREDENTIALS
./google-cloud-sdk/bin/gsutil cp $DB_URL /recorder.db

exec /flightreplay
