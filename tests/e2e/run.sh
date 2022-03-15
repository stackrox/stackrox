#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Runs all e2e tests. Derived from the workload of CircleCI gke-api-nongroovy-tests.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

source "$TEST_ROOT/scripts/lib.sh"
source "$TEST_ROOT/scripts/ci/sensor-wait.sh"
source "$TEST_ROOT/tests/scripts/setup-certs.sh"
source "$TEST_ROOT/tests/e2e/lib.sh"

test_e2e() {
    info "Starting test"

    DEPLOY_DIR="deploy/k8s"
    QUAY_REPO="rhacs-eng"
    if is_CI; then
        REGISTRY="quay.io/$QUAY_REPO"
    else
        REGISTRY="stackrox"
    fi

    test_preamble
    remove_existing_stackrox_resources
    setup_default_TLS_certs
    "$TEST_ROOT/tests/complianceoperator/create.sh"

    info "Deploying central"
    "$TEST_ROOT/$DEPLOY_DIR/central.sh"
    get_central_basic_auth_creds
    wait_for_api
    setup_client_TLS_certs

    info "Deploying sensor"
    "$TEST_ROOT/$DEPLOY_DIR/sensor.sh"
    sensor_wait

    prepare_for_endpoints_test

    run_roxctl_tests
    run_roxctl_bats_tests "roxctl-test-output"

    info "E2E API tests"
    make -C tests

    setup_proxy_tests
    run_proxy_tests

    collect_and_check_stackrox_logs "/tmp/e2e-test-logs" "initial_phase"

    info "E2E destructive tests"
    make -C tests destructive-tests

    restore_56_1_backup
    wait_for_api

    info "E2E external backup tests"
    make -C tests external-backup-tests
}

test_preamble() {
    require_executable "roxctl"

    if ! is_CI; then
        require_environment "MAIN_IMAGE_TAG" "This is typically the output from 'make tag'"

        if [[ "$(roxctl version)" != "$MAIN_IMAGE_TAG" ]]; then
            die "There is a version mismatch between roxctl and MAIN_IMAGE_TAG. A version mismatch can cause the deployment script to use a container roxctl which can have issues in dev environments."
        fi
        pwds="$(pgrep -f 'port-forward' -c || true)"
        if [[ "$pwds" -gt 5 ]]; then
            die "There are many port-fowards probably left over from a previous run of this test."
        fi
        cleanup_proxy_tests
        export MAIN_TAG="$MAIN_IMAGE_TAG"
    else
        MAIN_TAG=$(make --quiet tag)
        export MAIN_TAG
    fi

    export MONITORING_SUPPORT=true
    export SCANNER_SUPPORT=true
    export LOAD_BALANCER=lb
    export ROX_PLAINTEXT_ENDPOINTS="8080,grpc@8081"
    export ROXDEPLOY_CONFIG_FILE_MAP="$TEST_ROOT/scripts/ci/endpoints/endpoints.yaml"
    export ROX_NETWORK_DETECTION_BASELINE_SIMULATION=true
    export ROX_NETWORK_DETECTION_BLOCKED_FLOWS=true
    export SENSOR_HELM_DEPLOY=true
    export ROX_ACTIVE_VULN_MANAGEMENT=true
    export ROX_ACTIVE_VULN_REFRESH_INTERVAL=1m
    MONITORING_IMAGE="$REGISTRY/monitoring:$(cat "$TEST_ROOT"/MONITORING_VERSION)"
    export MONITORING_IMAGE
    SCANNER_IMAGE="$REGISTRY/scanner:$(cat "$TEST_ROOT"/SCANNER_VERSION)"
    export SCANNER_IMAGE
    SCANNER_DB_IMAGE="$REGISTRY/scanner-db:$(cat "$TEST_ROOT"/SCANNER_VERSION)"
    export SCANNER_DB_IMAGE

    export TRUSTED_CA_FILE="$TEST_ROOT/tests/bad-ca/untrusted-root-badssl-com.pem"
}

prepare_for_endpoints_test() {
    info "Preparation for endpoints_test.go"

    local gencerts_dir
    gencerts_dir="$(mktemp -d)"
    setup_client_CA_auth_provider
    setup_generated_certs_for_test "$gencerts_dir"
    patch_resources_for_test
    export SERVICE_CA_FILE="$gencerts_dir/ca.pem"
    export SERVICE_CERT_FILE="$gencerts_dir/sensor-cert.pem"
    export SERVICE_KEY_FILE="$gencerts_dir/sensor-key.pem"
    start_port_forwards_for_test
}

run_roxctl_bats_tests() {
    local output="${1}"
    local suite="${2}"
    if (( $# != 2 )); then
      die "Error: run_roxctl_bats_tests requires 2 arguments: run_roxctl_bats_tests <test_output> <suite>"
    fi
    [[ -d "$TEST_ROOT/tests/roxctl/bats-tests/$suite" ]] || die "Cannot find directory: $TEST_ROOT/tests/roxctl/bats-tests/$suite"

    info "Running Bats e2e tests on development roxctl"
    "$TEST_ROOT/tests/roxctl/bats-runner.sh" "$output" "$TEST_ROOT/tests/roxctl/bats-tests/$suite"
}

run_roxctl_tests() {
    info "Run roxctl tests"

    "$TEST_ROOT/tests/roxctl/token-file.sh"
    "$TEST_ROOT/tests/roxctl/slim-collector.sh"
    "$TEST_ROOT/tests/roxctl/authz-trace.sh"
    "$TEST_ROOT/tests/roxctl/istio-support.sh"
    "$TEST_ROOT/tests/roxctl/helm-chart-generation.sh"
    CA="$SERVICE_CA_FILE" "$TEST_ROOT/tests/yamls/roxctl_verification.sh"
}

setup_proxy_tests() {
    info "Setup for proxy tests"

    PROXY_CERTS_DIR="$(mktemp -d)"
    export PROXY_CERTS_DIR="$PROXY_CERTS_DIR"
    "$TEST_ROOT/scripts/ci/proxy/deploy.sh"

    # Try preventing kubectl port-forward from hitting the FD limit, see
    # https://github.com/kubernetes/kubernetes/issues/74551#issuecomment-910520361
    # Note: this might fail if we don't have the correct privileges. Unfortunately,
    # we cannot `sudo ulimit` because it is a shell builtin.
    ulimit -n 65535 || true

    nohup kubectl -n proxies port-forward svc/nginx-proxy-plain-http 10080:80 </dev/null &>/dev/null &
    nohup kubectl -n proxies port-forward svc/nginx-proxy-tls-multiplexed 10443:443 </dev/null &>/dev/null &
    nohup kubectl -n proxies port-forward svc/nginx-proxy-tls-multiplexed-tls-be 11443:443 </dev/null &>/dev/null &
    nohup kubectl -n proxies port-forward svc/nginx-proxy-tls-http1 12443:443 </dev/null &>/dev/null &
    nohup kubectl -n proxies port-forward svc/nginx-proxy-tls-http1-plain 13443:443 </dev/null &>/dev/null &
    nohup kubectl -n proxies port-forward svc/nginx-proxy-tls-http2 14443:443 </dev/null &>/dev/null &
    nohup kubectl -n proxies port-forward svc/nginx-proxy-tls-http2-plain 15443:443 </dev/null &>/dev/null &
    sleep 1

    if ! grep central-proxy.stackrox.local /etc/hosts; then
        sudo bash -c 'echo "127.0.0.1 central-proxy.stackrox.local" >>/etc/hosts'
    fi
}

cleanup_proxy_tests() {
    if kubectl get ns proxies; then
        kubectl delete ns proxies --wait
    fi
}

run_proxy_tests() {
    info "Running proxy tests"

    info "Test HTTP access to plain HTTP proxy"
    # --retry-connrefused only works when forcing IPv4, see https://github.com/appropriate/docker-curl/issues/5
    local license_status
    license_status="$(curl --retry 5 --retry-connrefused -4 --retry-delay 1 --retry-max-time 10 -f 'http://central-proxy.stackrox.local:10080/v1/metadata' | jq -r '.licenseStatus')"
    echo "Got license status ${license_status} from server"
    [[ "$license_status" == "VALID" ]]

    info "Test HTTPS access to multiplexed TLS proxy"
    # --retry-connrefused only works when forcing IPv4, see https://github.com/appropriate/docker-curl/issues/5
    license_status="$(
        curl --cacert "${PROXY_CERTS_DIR}/ca.crt" \
        --retry 5 --retry-connrefused -4 --retry-delay 1 --retry-max-time 10 \
        -f \
        'https://central-proxy.stackrox.local:10443/v1/metadata' | jq -r '.licenseStatus')"
    echo "Got license status ${license_status} from server"
    [[ "$license_status" == "VALID" ]]

    info "Test roxctl access to proxies"
    local proxies=(
        "Plaintext proxy:10080:plaintext"
        "Multiplexed TLS proxy with plain backends:10443"
        "Multiplexed TLS proxy with TLS backends:11443"
        "Multiplexed TLS proxy with plain backends (direct gRPC):10443:direct"
        "Multiplexed TLS proxy with TLS backends (direct gRPC):11443:direct"
        "HTTP/1 proxy with TLS backends:12443"
        "HTTP/1 proxy with plain backends:13443"
        "HTTP/2 proxy with TLS backends:14443"
        "HTTP/2 proxy with plain backends:15443"
    )

    local failures=()
    for p in "${proxies[@]}"; do
        local name
        name="$(echo "$p" | cut -d: -f1)"
        local port
        port="$(echo "$p" | cut -d: -f2)"
        local opt
        opt="$(echo "$p" | cut -d: -f3)"
        mkdir -p "/tmp/proxy-test-${port}-${opt}" && cd "/tmp/proxy-test-${port}-${opt}"

        local extra_args=()
        local scheme="https"
        local plaintext="false"
        local plaintext_neg="true"
        local direct=0
        case "$opt" in
        plaintext)
            extra_args=(--insecure)
            plaintext="true"
            plaintext_neg="false"
            scheme="http"
            ;;
        direct)
            extra_args=(--direct-grpc)
            direct=1
            ;;
        esac

        info "Testing roxctl access through ${name}..."
        local endpoint="central-proxy.stackrox.local:${port}"
        for endpoint_tgt in "${scheme}://${endpoint}" "${scheme}://${endpoint}/" "$endpoint"; do
        roxctl "${extra_args[@]}" --plaintext="$plaintext" -e "${endpoint_tgt}" -p "$ROX_PASSWORD" central debug log >/dev/null || \
            failures+=("$p")

        if (( direct )); then
            roxctl "${extra_args[@]}" --plaintext="$plaintext" --force-http1 -e "${endpoint_tgt}" -p "$ROX_PASSWORD" central debug log &>/dev/null && \
            failures+=("${p},force-http1")
        else
            roxctl "${extra_args[@]}" --plaintext="$plaintext" --force-http1 -e "${endpoint_tgt}" -p "$ROX_PASSWORD" central debug log >/dev/null || \
            failures+=("${p},force-http1")
        fi

        if [[ "$endpoint_tgt" = *://* ]]; then
            # Auto-sense plaintext or TLS when specifying a scheme
            roxctl "${extra_args[@]}" -e "${endpoint_tgt}" -p "$ROX_PASSWORD" central debug log >/dev/null || \
            failures+=("${p},tls-autosense")

            # Incompatible plaintext configuration should fail
            roxctl "${extra_args[@]}" --plaintext="$plaintext_neg" -e "${endpoint_tgt}" -p "$ROX_PASSWORD" central debug log &>/dev/null && \
            failures+=("${p},incompatible-tls")
        fi

        done
        roxctl "${extra_args[@]}" --plaintext="$plaintext" -e "central-proxy.stackrox.local:${port}" -p "$ROX_PASSWORD" sensor generate k8s --name remote --continue-if-exists || \
        failures+=("${p},sensor-generate")
        echo "Done."
        rm -rf "/tmp/proxy-test-${port}"
    done

    echo "Total: ${#failures[@]} failures."
    if (( ${#failures[@]} > 0 )); then
        printf " - %s\n" "${failures[@]}"
        exit 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_e2e "$*"
fi
