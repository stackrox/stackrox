#!/usr/bin/env bash

set -euo pipefail

# A library or reusable bash functions

usage() {
    echo "lib.sh provides a library or reusable bash functions.
Invoke with:
  $ scripts/lib.sh method [args...]
Reuse with:
  . scripts/lib.sh
  method [args...]"
}

info() {
    echo "INFO: $(date): $*"
}

die() {
    echo >&2 "$@"
    exit 1
}

is_CI() {
    (
        set +u
        [[ -n "$CI" || -n "$CIRCLECI" ]]
    )
}

is_darwin() {
    uname -a | grep -i darwin >/dev/null 2>&1
}

is_linux() {
    uname -a | grep -i linux >/dev/null 2>&1
}

require_environment() {
    (
        set +u
        if [[ -z "$(eval echo "\$$1")" ]]; then
            die "missing $1 environment variable"
        fi
    )
}

require_executable() {
    if ! command -v "$1" >/dev/null 2>&1; then
        die "missing $1 executable"
    fi
}

check_stackrox_logs() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: check_stackrox_logs <dir>"
    fi

    local dir="$1"

    if [[ ! -d "$dir/stackrox/pods" ]]; then
        die "StackRox logs were not collected. (See ./scripts/ci/collect-service-logs.sh stackrox)"
    fi

    local previous_logs
    previous_logs=$(ls "$dir"/stackrox/pods/*-previous.log || true)
    if [[ -n "$previous_logs" ]]; then
        echo >&2 "Previous logs found"
        # shellcheck disable=SC2086
        if ! scripts/ci/logcheck/check-restart-logs.sh upgrade-tests $previous_logs; then
            exit 1
        fi
    fi

    local logs
    logs=$(ls "$dir"/stackrox/pods/*.log)
    local filtered
    # shellcheck disable=SC2010,SC2086
    filtered=$(ls $logs | grep -v "previous.log" || true)
    if [[ -n "$filtered" ]]; then
        # shellcheck disable=SC2086
        if ! scripts/ci/logcheck/check.sh $filtered; then
            die "Found at least one suspicious log file entry."
        fi
    fi
}

remove_existing_stackrox_resources() {
    info "Will remove any existing stackrox resources"

    kubectl -n stackrox delete cm,deploy,ds,networkpolicy,secret,svc,serviceaccount,validatingwebhookconfiguration,pv,pvc,clusterrole,clusterrolebinding,role,rolebinding,psp -l "app.kubernetes.io/name=stackrox" --wait || true
    # openshift specific:
    kubectl -n stackrox delete SecurityContextConstraints -l "app.kubernetes.io/name=stackrox" --wait || true
    kubectl delete -R -f scripts/ci/psp --wait || true
    kubectl delete ns stackrox --wait || true
    helm uninstall monitoring || true
    helm uninstall central || true
    helm uninstall scanner || true
    helm uninstall sensor || true
    kubectl get namespace -o name | grep -E '^namespace/qa' | xargs kubectl delete --wait || true
}

wait_for_api() {
    info "Waiting for central to start"

    start_time="$(date '+%s')"
    while true; do
        central_json="$(kubectl -n stackrox get deploy/central -o json)"
        if [[ "$(echo "${central_json}" | jq '.status.replicas')" == 1 && "$(echo "${central_json}" | jq '.status.readyReplicas')" == 1 ]]; then
            break
        fi
        if (($(date '+%s') - start_time > 300)); then
            kubectl -n stackrox get pod -o wide
            kubectl -n stackrox get deploy -o wide
            echo >&2 "Timed out after 5m"
            exit 1
        fi
        echo -n .
        sleep 1
    done

    echo "Central is running"

    info "Waiting for centrals API"

    API_HOSTNAME=localhost
    API_PORT=8000
    LOAD_BALANCER="${LOAD_BALANCER:-}"
    if [[ "${LOAD_BALANCER}" == "lb" ]]; then
        API_HOSTNAME=$(./scripts/k8s/get-lb-ip.sh)
        API_PORT=443
    fi
    API_ENDPOINT="${API_HOSTNAME}:${API_PORT}"
    METADATA_URL="https://${API_ENDPOINT}/v1/metadata"
    echo "METADATA_URL is set to ${METADATA_URL}"
    set +e
    NUM_SUCCESSES_IN_A_ROW=0
    SUCCESSES_NEEDED_IN_A_ROW=3
    # shellcheck disable=SC2034
    for i in $(seq 1 40); do
        metadata="$(curl -sk --connect-timeout 5 --max-time 10 "${METADATA_URL}")"
        metadata_exitstatus="$?"
        status="$(echo "$metadata" | jq '.licenseStatus' -r)"
        if [[ "$metadata_exitstatus" -eq "0" && "$status" != "RESTARTING" ]]; then
            NUM_SUCCESSES_IN_A_ROW=$((NUM_SUCCESSES_IN_A_ROW + 1))
            if [[ "${NUM_SUCCESSES_IN_A_ROW}" == "${SUCCESSES_NEEDED_IN_A_ROW}" ]]; then
                break
            fi
            echo "Status is now: ${status}"
            sleep 2
            continue
        fi
        NUM_SUCCESSES_IN_A_ROW=0
        echo -n .
        sleep 5
    done
    echo
    if [[ "${NUM_SUCCESSES_IN_A_ROW}" != "${SUCCESSES_NEEDED_IN_A_ROW}" ]]; then
        kubectl -n stackrox get pod
        echo "Failed to connect to Central. Failed with ${NUM_SUCCESSES_IN_A_ROW} successes in a row"
        exit 1
    fi
    set -e

    export API_HOSTNAME="${API_HOSTNAME}"
    export API_PORT="${API_PORT}"
    export API_ENDPOINT="${API_ENDPOINT}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        usage
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$*"
fi
