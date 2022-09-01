#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Test utility functions for upgrades

wait_for_central_reconciliation() {
    info "Waiting for central reconciliation"

    # Reconciliation is rather slow in this case, since the central has a DB with a bunch of deployments,
    # none of which exist. So when sensor connects, the reconciliation deletion takes a while to flush.
    # This causes flakiness with the smoke tests.
    # To mitigate this, wait for the deployments to get deleted before running the tests.
    local success=0
    for i in $(seq 1 90); do
        local numDeployments
        numDeployments="$(curl -sSk -u "admin:$ROX_PASSWORD" "https://$API_ENDPOINT/v1/summary/counts" | jq '.numDeployments' -r)"
        echo "Try number ${i}. Number of deployments in Central: $numDeployments"
        [[ -n "$numDeployments" ]]
        if [[ "$numDeployments" -lt 100 ]]; then
            success=1
            break
        fi
        sleep 10
    done
    [[ "$success" == 1 ]]
}

wait_for_scanner_to_be_ready() {
    echo "Waiting for scanner to be ready"
    start_time="$(date '+%s')"
    while true; do
      scanner_json="$(kubectl -n stackrox get deploy/scanner -o json)"
      replicas="$(jq '.status.replicas' <<<"${scanner_json}")"
      readyReplicas="$(jq '.status.readyReplicas' <<<"${scanner_json}")"
      echo "scanner replicas: $replicas"
      echo "scanner readyReplicas: $readyReplicas"
      if [[  "$replicas" == "$readyReplicas" ]]; then
        break
      fi
      if (( $(date '+%s') - start_time > 300 )); then
        kubectl -n stackrox get pod -o wide
        kubectl -n stackrox get deploy -o wide
        echo >&2 "Timed out after 5m"
        exit 1
      fi
      sleep 5
    done
    echo "Scanner is ready"
}

validate_upgrade() {
    if [[ "$#" -ne 3 ]]; then
        die "missing args. usage: validate_upgrade <stage name> <stage description> <upgrade_cluster_id>"
    fi

    local stage_name="$1"
    local stage_description="$2"
    local upgrade_cluster_id="$3"
    local policies_dir="../pkg/defaults/policies/files"

    if [ -n $API_TOKEN ]; then
        info "Verifying API token generated can access the central"
        echo $API_TOKEN | $TEST_ROOT/bin/$TEST_HOST_OS/roxctl --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" --token-file /dev/stdin central whoami > /dev/null
    fi

    info "Validating the upgrade with upgrade tests: $stage_description"

    CLUSTER="$CLUSTER_TYPE_FOR_TEST" \
        UPGRADE_CLUSTER_ID="$upgrade_cluster_id" \
        POLICIES_JSON_RELATIVE_PATH="$policies_dir" \
        make -C qa-tests-backend upgrade-test || touch FAIL
    store_qa_test_results "validate-upgrade-tests-${stage_name}"
    [[ ! -f FAIL ]] || die "Upgrade tests failed"
}

function roxcurl() {
  local url="$1"
  shift
  curl -u "admin:${ROX_PASSWORD}" -k "https://${API_ENDPOINT}${url}" "$@"
}
