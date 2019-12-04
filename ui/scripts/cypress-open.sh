#!/usr/bin/env bash

# Opens cypress with environment variables for feature flags and auth
api_endpoint="${UI_BASE_URL:-https://localhost:8000}"

if [[ -z "$ROX_USERNAME" || -z "$ROX_PASSWORD" ]]; then
  # basic auth creds weren't set (e.g. by CI), assume local k8s deployment
  source ../scripts/k8s/export-basic-auth-creds.sh ../deploy/k8s
fi

if [[ -n "$ROX_PASSWORD" ]]; then
  readarray -t arr < <(curl -sk -u admin:$ROX_PASSWORD ${api_endpoint}/v1/featureflags | jq -cr '.featureFlags[] | {name: .envVar, enabled: .enabled}')
  for i in "${arr[@]}"; do
    name=$(echo $i | jq -rc .name)
    val=$(echo $i | jq -rc .enabled)
    export CYPRESS_${name}=${val}
  done
fi
export CYPRESS_ROX_AUTH_TOKEN=$(./scripts/get-auth-token.sh)
cypress open
