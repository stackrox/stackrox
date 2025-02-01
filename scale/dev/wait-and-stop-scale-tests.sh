#!/usr/bin/env bash
set -eou pipefail

if [[ -z "${API_ENDPOINT:-}" ]]; then
  echo "API_ENDPOINT must be set"
  exit 1
fi

if [[ -z "${ROX_PASSWORD:-}" ]]; then
  echo "ROX_PASSWORD must be set"
  exit 1
fi

max_deployments=${1:-}

kill_sensors() {
  echo "Killing sensors"
  mapfile -t sensor_namespaces < <(kubectl get ns -o custom-columns=:metadata.name | grep -E 'stackrox[0-9]+')
  nnamespace="${#sensor_namespaces[@]}"
  for ((i = 0; i < nnamespace; i = i + 1)); do
    #kubectl delete ns "${sensor_namespaces[i]}"
    echo "Deleting namespace ${sensor_namespaces[i]}"
  done
}

set_api_token() {
  ROX_BASE_URL="https://$API_ENDPOINT"
  target_url="$(curl -sSkf -u "admin:${ROX_PASSWORD}" -o /dev/null -w '%{redirect_url}' "${ROX_BASE_URL}/sso/providers/basic/4df1b98c-24ed-4073-a9ad-356aec6bb62d/challenge?micro_ts=0")"
  
  ROX_API_TOKEN="$(echo "$target_url" | sed 's|.*token=||' | sed 's|&type.*||')"
}

set_api_token

while true; do
  deployment_count="$(curl --location --silent --request GET "https://${API_ENDPOINT}/v1/deploymentscount" -k --header "Authorization: Bearer $ROX_API_TOKEN" | jq .count)"
  if [[ "$deployment_count" -gt "$max_deployments" ]]; then
     echo "The number of deployments, $deployment_count, is greater than the maximum number of deployments, $max_deployments."
     kill_sensors
     exit 0
  fi
  sleep 5
done
