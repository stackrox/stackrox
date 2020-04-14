#!/usr/bin/env bash

# refresh token every 15m
while sleep 900; do
	gcloud config config-helper --force-auth-refresh >/dev/null
	cmd=(gcloud container clusters get-credentials --project stackrox-ci --zone "$ZONE" "$CLUSTER_NAME")
done
