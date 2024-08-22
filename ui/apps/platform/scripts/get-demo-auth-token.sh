#!/usr/bin/env bash

# Makes a request to create a new auth token that is printed to stdout.
# Env vars ROX_USERNAME and ROX_PASSWORD are expected to be set for basic auth

api_endpoint="${UI_BASE_URL:-https://localhost:8000}"

if [[ -z "$CYPRESS_DEMO_PASSWORD" ]]; then
    echo >&2 "Expected CYPRESS_DEMO_PASSWORD env var for basic auth creds. Please set the env var with the Central Password"
    exit 1
fi

if [[ -z "$ROX_USERNAME" || -z "$ROX_PASSWORD" ]]; then
    # basic auth creds weren't set (e.g. by CI), assume local k8s deployment
    export ROX_USERNAME=admin
    export ROX_PASSWORD=$CYPRESS_DEMO_PASSWORD
fi

curl_cfg() { # Use built-in echo to not expose $2 in the process list.
    echo -n "$1 = \"${2//[\"\\]/\\&}\""
}

if [[ -n "$ROX_USERNAME" && -n "$ROX_PASSWORD" ]]; then
  rox_auth_token="$(
  curl -sk --config <(curl_cfg user "${ROX_USERNAME}:${ROX_PASSWORD}") \
    "${api_endpoint}/v1/apitokens/generate" \
    -X POST \
    -d '{"name": "ui_demo_tests", "role": "Admin"}' \
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
