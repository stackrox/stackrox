#!/usr/bin/env bash
set -eoux pipefail

if [[ -z "${API_ENDPOINT:-}" ]]; then
  echo "API_ENDPOINT must be set"
  exit 1
fi

if [[ -z "${ROX_PASSWORD:-}" ]]; then
  echo "ROX_PASSWORD must be set"
  exit 1
fi

max_deployments=${1:-30000}

while true; do
  deployment_count="$(curl --location --silent --user "admin:${ROX_PASSWORD}" --request GET "https://${API_ENDPOINT}/v1/deploymentscount" -k | jq .count)"
  if [[ "$deployment_count" -gt "$max_deployments" ]]; then
     echo "The number of deployments, $deployment_count, is greater than the maximum number of deployments, $max_deployments."
     exit 0
  fi
  echo "${deployment_count} deployments in Central. Waiting for total ${max_deployments}"
  sleep 30
done
