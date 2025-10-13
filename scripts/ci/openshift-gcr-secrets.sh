#! /bin/bash

oc create namespace qa
oc project qa
oc create secret docker-registry --docker-username=_json_key --docker-password="$GOOGLE_CREDENTIALS_GCR_SCANNER_V2" --docker-server https://us.gcr.io --docker-email stackrox@stackrox.com gcr
oc secrets link serviceaccount/default secrets/gcr --for=pull
