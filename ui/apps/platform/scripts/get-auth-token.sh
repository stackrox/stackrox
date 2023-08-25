#!/usr/bin/env bash

# Makes a request to create a new auth token that is printed to stdout.
# Env vars ROX_USERNAME and ROX_PASSWORD are expected to be set for basic auth,
# otherwise k8s deployment assumed from where basic auth creds are retrieved.

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

api_endpoint="${UI_BASE_URL:-https://localhost:8000}"

if [[ -z "$ROX_USERNAME" || -z "$ROX_PASSWORD" ]]; then
  # basic auth creds weren't set (e.g. by CI), assume local k8s deployment
  source "${DIR}/../../../../scripts/k8s/export-basic-auth-creds.sh" "${DIR}/../../../../deploy/k8s"
fi

if [[ -n "$ROX_USERNAME" && -n "$ROX_PASSWORD" ]]; then
  rox_auth_token="$(
  curl -sk -u "${ROX_USERNAME}:${ROX_PASSWORD}" "${api_endpoint}/v1/apitokens/generate" \
    -X POST \
    -d '{"name": "ui_tests", "role": "Admin"}' \
    | jq -r '.token // ""')"
else
  echo >&2 "Expected ROX_USERNAME and ROX_PASSWORD env vars for basic auth creds"
  exit 1
fi

if [[ -z "$rox_auth_token" ]]; then
  echo >&2 "Could not issue an auth token"
  exit 1
fi

echo $rox_auth_token
