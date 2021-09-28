#! /usr/bin/env bash

set -uo pipefail

# This test script requires API_ENDPOINT and ROX_PASSWORD to be set in the environment.

[[ -n "$API_ENDPOINT" ]] || die "API_ENDPOINT environment variable required"
[[ -n "$ROX_PASSWORD" ]] || die "ROX_PASSWORD environment variable required"

echo "Testing command: roxctl central debug authz-trace"

eecho() {
  echo "$@" >&2
}

die() {
  eecho "$@"
  exit 1
}

curl_central() {
  url="$1"
  shift
  [[ -n "${url}" ]] || die "No URL specified"
  curl -Sskf "https://${API_ENDPOINT}${url}" "$@"
}

curl_central_admin() {
  curl_central "$@" -u "admin:${ROX_PASSWORD}"
}

curl_central_token() {
  curl_central "$@" -H "Authorization: Bearer $(cat $TOKEN_FILE)"
}

TARGET_ENDPOINT="/v1/alertscount"

# Retrieve API token.
echo "Retrieve API token"
TOKEN_FILE=$(mktemp)
curl_central_admin /v1/apitokens/generate -d '{"name": "test", "roles": ["Analyst"]}' | jq -r .token > "$TOKEN_FILE"
[[ -n "$(cat "$TOKEN_FILE")" ]] || die "Failed to retrieve API token"

FAILED_CHECKS=0

# Run authorization trace collection in the background.
nohup roxctl --endpoint "$API_ENDPOINT" --insecure-skip-tls-verify --insecure -p "$ROX_PASSWORD" -t 10s central debug authz-trace > trace.out &
# Wait for roxctl to subscribe for authz traces.
sleep 5

# Query Central to get a specific authz trace.
curl_central_token $TARGET_ENDPOINT -o /dev/null || die "Failed to query alerts count"

# Wait for a record triggered by the request to appear in the trace file.
( tail -f trace.out & ) | grep -q $TARGET_ENDPOINT

# Extract the trace triggered by the request.
log_level_trace="$(jq <trace.out -cr 'select(.request.endpoint == "'$TARGET_ENDPOINT'") | .trace' | head -n 1)"

# Check if the trace is not null.
if [[ -z "$log_level_trace" || "$log_level_trace" == "null" ]]; then
  eecho "Expected trace for '$TARGET_ENDPOINT' not found"
  FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi
# Check if scopeCheckerType is built-in.
scope_checker_type=$(echo "$log_level_trace" | jq -r '.scopeCheckerType')
if [ "$scope_checker_type" != "built-in" ]; then
  eecho "scopeCheckerType should be set to 'built-in'"
  FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi
# Check clustersTotalNum > 0.
clusters_total_num=$(echo "$log_level_trace" | jq -r '.builtIn.clustersTotalNum')
if [ "$clusters_total_num" -le "0" ]; then
  eecho "clustersTotalNum should be > 0"
  FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi
# Check namespaceTotalNum > 0.
namespaces_total_num=$(echo "$log_level_trace" | jq -r '.builtIn.namespacesTotalNum')
if [ "$namespaces_total_num" -le "0" ]; then
  eecho "namespacesTotalNum should be > 0"
  FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

if (( FAILED_CHECKS == 0 )); then
  echo "Passed"
else
  eecho "$FAILED_CHECKS checks failed"
  eecho "Trace:"
  echo "$log_level_trace" | jq . >&2
  exit 1
fi
