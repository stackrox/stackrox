#! /usr/bin/env bash

set -uo pipefail

# This test script requires API_ENDPOINT and ROX_ADMIN_PASSWORD to be set in the environment.

[[ -n "$API_ENDPOINT" ]] || die "API_ENDPOINT environment variable required"
[[ -n "$ROX_ADMIN_PASSWORD" ]] || die "ROX_ADMIN_PASSWORD environment variable required"

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
  curl -Sskf --retry 5 --retry-connrefused "https://${API_ENDPOINT}${url}" "$@"
}

curl_cfg() { # Use built-in echo to not expose $2 in the process list.
  echo -n "$1 = \"${2//[\"\\]/\\&}\""
}

curl_central_admin() {
  curl_central "$@" --config <(curl_cfg user "admin:${ROX_ADMIN_PASSWORD}")
}

curl_central_token() {
  curl_central "$@" --config <(curl_cfg header "Authorization: Bearer $(cat "$TOKEN_FILE")")
}

verify_trace_for_endpoint() {
  echo "Verifying trace for endpoint: " "$@"
  target_endpoint="$@"
  # Wait for a record triggered by the request to appear in the trace file.
  ( tail -f trace.out & ) | grep -q $target_endpoint

  # Extract the trace triggered by the request.
  target_trace="$(jq <trace.out -cr 'select(.request.endpoint == "'$target_endpoint'") | .trace' | head -n 1)"

  # Check if the trace is not null.
  if [[ -z "$target_trace" || "$target_trace" == "null" ]]; then
    eecho "Expected trace for '$target_endpoint' not found"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
  fi
  # Check if scopeCheckerType is built-in.
  scope_checker_type=$(echo "$target_trace" | jq -r '.scopeCheckerType')
  if [ "$scope_checker_type" != "built-in" ]; then
    eecho "scopeCheckerType should be set to 'built-in'"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
  fi
  # Check clustersTotalNum > 0.
  clusters_total_num=$(echo "$target_trace" | jq -r '.builtIn.clustersTotalNum')
  if [ "$clusters_total_num" -le "0" ]; then
    eecho "clustersTotalNum should be > 0"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
  fi
  # Check namespaceTotalNum > 0.
  namespaces_total_num=$(echo "$target_trace" | jq -r '.builtIn.namespacesTotalNum')
  if [ "$namespaces_total_num" -le "0" ]; then
    eecho "namespacesTotalNum should be > 0"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
  fi

  if (( FAILED_CHECKS == 0 )); then
    echo "Passed for endpoint: " "$@"
  else
    eecho "$FAILED_CHECKS checks failed"
    eecho "Trace:"
    echo "$target_trace" | jq . >&2
    exit 1
  fi
}

# Retrieve API token.
echo "Retrieve API token"
TOKEN_FILE=$(mktemp)
curl_central_admin /v1/apitokens/generate -d '{"name": "test", "roles": ["Analyst"]}' | jq -r .token > "$TOKEN_FILE"
[[ -n "$(cat "$TOKEN_FILE")" ]] || die "Failed to retrieve API token"

FAILED_CHECKS=0

# Run authorization trace collection in the background.
nohup roxctl --endpoint "$API_ENDPOINT" --ca "" --insecure-skip-tls-verify --insecure -t 20s central debug authz-trace > trace.out &
# Wait for roxctl to subscribe for authz traces.
sleep 5

# Query Central to get a specific non-GRPC authz trace.
REQUEST_PAYLOAD='{"operationName":"getNodes","variables":{"query":""},"query":"query getNodes($query: String) {nodeCount(query: $query)}"}'
curl_central_token "/api/graphql?opname=getNodes" -d "$REQUEST_PAYLOAD" -X POST -o /dev/null || die "Failed to query GraphQL getNodes"
verify_trace_for_endpoint "/api/graphql?opname=getNodes"

# Query Central to get a specific GRPC authz trace.
curl_central_token "/v1/alertscount" -o /dev/null || die "Failed to query alerts count"
verify_trace_for_endpoint "/v1/alertscount"
