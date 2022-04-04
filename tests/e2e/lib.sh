#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Test utility functions

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

source "$TEST_ROOT/scripts/lib.sh"
source "$TEST_ROOT/scripts/ci/lib.sh"

get_central_basic_auth_creds() {
    info "Getting central basic auth creds"

    require_environment "TEST_ROOT"
    require_environment "DEPLOY_DIR"

    source "$TEST_ROOT/scripts/k8s/export-basic-auth-creds.sh" "$DEPLOY_DIR"

    ci_export "ROX_USERNAME" "$ROX_USERNAME"
    ci_export "ROX_PASSWORD" "$ROX_PASSWORD"
}

setup_client_CA_auth_provider() {
    info "Set up client CA auth provider for endpoints_test.go"

    require_environment "API_ENDPOINT"
    require_environment "ROX_PASSWORD"
    require_environment "CLIENT_CA_PATH"

    roxctl -e "$API_ENDPOINT" -p "$ROX_PASSWORD" \
        central userpki create test-userpki -r Analyst -c "$CLIENT_CA_PATH"
}

setup_generated_certs_for_test() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: setup_generated_certs_for_test <dir>"
    fi

    info "Setting up generated certs for test"

    local dir="$1"

    require_environment "API_ENDPOINT"
    require_environment "ROX_PASSWORD"

    roxctl -e "$API_ENDPOINT" -p "$ROX_PASSWORD" \
        sensor generate-certs remote --output-dir "$dir"
    [[ -f "$dir"/cluster-remote-tls.yaml ]]
    # Use the certs in future steps that will use client auth.
    # This will ensure that the certs are valid.
    sensor_tls_cert="$(kubectl create --dry-run=client -o json -f "$dir"/cluster-remote-tls.yaml | jq 'select(.metadata.name=="sensor-tls")')"
    for file in ca.pem sensor-cert.pem sensor-key.pem; do
        echo "${sensor_tls_cert}" | jq --arg filename "${file}" '.stringData[$filename]' -r > "$dir/${file}"
    done
}

patch_resources_for_test() {
    info "Patch the loadbalancer and netpol resources for endpoints test"

    require_environment "TEST_ROOT"
    require_environment "API_HOSTNAME"

    kubectl -n stackrox patch svc central-loadbalancer --patch "$(cat "$TEST_ROOT"/.circleci/endpoints-test-lb-patch.yaml)"
    kubectl -n stackrox apply -f "$TEST_ROOT/.circleci/endpoints-test-netpol.yaml"
    # shellcheck disable=SC2034
    for i in $(seq 1 20); do
        if curl -sk --fail "https://${API_HOSTNAME}:8446/v1/metadata" &>/dev/null; then
            return
        fi
        sleep 1
    done
    die "Port 8446 did not become reachable in time"
    exit 1
}

start_port_forwards_for_test() {
    info "Creating port-forwards for test"

    # Try preventing kubectl port-forward from hitting the FD limit, see
    # https://github.com/kubernetes/kubernetes/issues/74551#issuecomment-910520361
    # Note: this might fail if we don't have the correct privileges. Unfortunately,
    # we cannot `sudo ulimit` because it is a shell builtin.
    ulimit -n 65535 || true

    central_pod="$(kubectl -n stackrox get po -lapp=central -oname | head -n 1)"
    for target_port in 8080 8081 8082 8443 8444 8445 8446 8447 8448; do
        nohup kubectl -n stackrox port-forward "${central_pod}" "$((target_port + 10000)):${target_port}" </dev/null &>/dev/null &
    done
    sleep 1
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

collect_and_check_stackrox_logs() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: collect_and_check_stackrox_logs <output-dir> <test_stage>"
    fi

    local dir="$1/$2"

    info "Will collect stackrox logs to $dir and check them"

    ./scripts/ci/collect-service-logs.sh stackrox "$dir"

    check_stackrox_logs "$dir"
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

# When working as expected it takes less than one minute for the API server to
# reach ready. Often times out on OSD. If this call fails in CI we need to
# identify the source of pull/scheduling latency, request throttling, etc.
# I tried increasing the timeout from 5m to 20m for OSD but it did not help.
wait_for_api() {
    info "Waiting for Central to start"

    start_time="$(date '+%s')"
    max_seconds=300

    while true; do
        central_json="$(kubectl -n stackrox get deploy/central -o json)"
        replicas="$(jq '.status.replicas' <<<"$central_json")"
        ready_replicas="$(jq '.status.readyReplicas' <<<"$central_json")"
        curr_time="$(date '+%s')"
        elapsed_seconds=$(( curr_time - start_time ))

        # Ready case
        if [[ "$replicas" == 1 && "$ready_replicas" == 1 ]]; then
            sleep 30
            break
        fi

        # Timeout case
        if (( elapsed_seconds > max_seconds )); then
            kubectl -n stackrox get pod -o wide
            kubectl -n stackrox get deploy -o wide
            echo >&2 "wait_for_api() timeout after $max_seconds seconds."
            exit 1
        fi

        # Otherwise report and retry
        echo "waiting ($elapsed_seconds/$max_seconds)"
        sleep 5
    done

    info "Central deployment is ready."
    info "Waiting for Central API endpoint"

    API_HOSTNAME=localhost
    API_PORT=8000
    LOAD_BALANCER="${LOAD_BALANCER:-}"
    if [[ "${LOAD_BALANCER}" == "lb" ]]; then
        API_HOSTNAME=$(./scripts/k8s/get-lb-ip.sh)
        API_PORT=443
    fi
    API_ENDPOINT="${API_HOSTNAME}:${API_PORT}"
    METADATA_URL="https://${API_ENDPOINT}/v1/metadata"
    info "METADATA_URL is set to ${METADATA_URL}"

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
            info "Status is now: ${status}"
            sleep 2
            continue
        fi
        NUM_SUCCESSES_IN_A_ROW=0
        echo -n .
        sleep 5
    done
    echo
    if [[ "${NUM_SUCCESSES_IN_A_ROW}" != "${SUCCESSES_NEEDED_IN_A_ROW}" ]]; then
        info "Failed to connect to Central. Failed with ${NUM_SUCCESSES_IN_A_ROW} successes in a row"
        info "port-forwards:"
        pgrep port-forward
        info "pods:"
        kubectl -n stackrox get pod
        exit 1
    fi
    set -e

    ci_export API_HOSTNAME "${API_HOSTNAME}"
    ci_export API_PORT "${API_PORT}"
    ci_export API_ENDPOINT "${API_ENDPOINT}"
}

restore_56_1_backup() {
    info "Restoring a 56.1 backup"

    require_environment "API_ENDPOINT"
    require_environment "ROX_PASSWORD"

    gsutil cp gs://stackrox-ci-upgrade-test-dbs/stackrox_56_1_fixed_upgrade.zip .
    roxctl -e "$API_ENDPOINT" -p "$ROX_PASSWORD" \
        central db restore --timeout 2m stackrox_56_1_fixed_upgrade.zip
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        usage
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
