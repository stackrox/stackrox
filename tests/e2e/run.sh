#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Runs all e2e tests. Derived from the workload of CircleCI gke-api-nongroovy-tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

# shellcheck source=../../scripts/lib.sh
source "$ROOT/scripts/lib.sh"
# shellcheck source=../../scripts/ci/sensor-wait.sh
source "$ROOT/scripts/ci/sensor-wait.sh"
# shellcheck source=../../tests/scripts/setup-certs.sh
source "$ROOT/tests/scripts/setup-certs.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"

test_e2e() {
    info "Starting e2e tests"

    require_environment "KUBECONFIG"

    export_test_environment
    # TODO(ROX-18827) turn off new compliance for now
    ci_export ROX_COMPLIANCE_ENHANCEMENTS "false"

    export SENSOR_HELM_DEPLOY=true
    export ROX_ACTIVE_VULN_REFRESH_INTERVAL=1m
    export ROX_NETPOL_FIELDS=true

    test_preamble
    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs
    "$ROOT/tests/complianceoperator/create.sh"

    deploy_stackrox
    deploy_optional_e2e_components

    rm -f FAIL

    prepare_for_endpoints_test

    run_roxctl_tests
    run_roxctl_bats_tests "roxctl-test-output" "cluster" || touch FAIL
    store_test_results "roxctl-test-output" "roxctl-test-output"
    [[ ! -f FAIL ]] || die "roxctl e2e tests failed"

    # Give some time for previous tests to finish up
    wait_for_api

    info "E2E API tests"
    if pr_has_label "ci-release-build"; then
        echo "Running e2e tests in release mode"
        export GOTAGS=release
    fi
    make -C tests || touch FAIL
    store_test_results "tests/all-tests-results" "all-tests-results"
    [[ ! -f FAIL ]] || die "e2e API tests failed"

    # Give some time for previous tests to finish up
    wait_for_api

    setup_proxy_tests "localhost"
    run_proxy_tests "localhost"
    cd "$ROOT"

    collect_and_check_stackrox_logs "/tmp/e2e-test-logs" "initial_tests"

    # Give some time for previous tests to finish up
    wait_for_api

    info "E2E destructive tests"
    make -C tests destructive-tests || touch FAIL
    store_test_results "tests/destructive-tests-results" "destructive-tests-results"
    [[ ! -f FAIL ]] || die "destructive e2e tests failed"

    # Give some time for previous tests to finish up
    wait_for_api
    restore_56_1_backup
    wait_for_api

    info "E2E external backup tests"
    make -C tests external-backup-tests || touch FAIL
    store_test_results "tests/external-backup-tests-results" "external-backup-tests-results"
    [[ ! -f FAIL ]] || die "external backup e2e tests failed"
}

test_preamble() {
    require_executable "roxctl"

    MAIN_TAG=$(make --quiet --no-print-directory tag)
    export MAIN_TAG

    export ROX_PLAINTEXT_ENDPOINTS="8080,grpc@8081"
    export ROXDEPLOY_CONFIG_FILE_MAP="$ROOT/scripts/ci/endpoints/endpoints.yaml"
    export TRUSTED_CA_FILE="$ROOT/tests/bad-ca/untrusted-root-badssl-com.pem"
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
}

run_roxctl_bats_tests() {
    local output="${1}"
    local suite="${2}"
    if (( $# != 2 )); then
      die "Error: run_roxctl_bats_tests requires 2 arguments: run_roxctl_bats_tests <test_output> <suite>"
    fi
    [[ -d "$ROOT/tests/roxctl/bats-tests/$suite" ]] || die "Cannot find directory: $ROOT/tests/roxctl/bats-tests/$suite"

    info "Running Bats e2e tests on development roxctl"
    "$ROOT/tests/roxctl/bats-runner.sh" "$output" "$ROOT/tests/roxctl/bats-tests/$suite"
}

run_roxctl_tests() {
    info "Run roxctl tests"

    junit_wrap "roxctl-token-file" "roxctl token-file test" "" \
        "$ROOT/tests/roxctl/token-file.sh"

    junit_wrap "roxctl-slim-collector" "roxctl slim-collector test" "" \
        "$ROOT/tests/roxctl/slim-collector.sh"

    junit_wrap "roxctl-authz-trace" "roxctl authz-trace test" "" \
        "$ROOT/tests/roxctl/authz-trace.sh"

    junit_wrap "roxctl-istio-support" "roxctl istio-support test" "" \
        "$ROOT/tests/roxctl/istio-support.sh"

    junit_wrap "roxctl-helm-chart-generation" "roxctl helm-chart-generation test" "" \
        "$ROOT/tests/roxctl/helm-chart-generation.sh"

    CA="$SERVICE_CA_FILE" junit_wrap "roxctl-yaml-verification" "roxctl yaml-verification test" "" \
        "$ROOT/tests/yamls/roxctl_verification.sh"
}

setup_proxy_tests() {
    info "Setup for proxy tests"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: setup_proxy_tests <server_name>"
    fi

    local server_name="$1"

    PROXY_CERTS_DIR="$(mktemp -d)"
    export PROXY_CERTS_DIR="$PROXY_CERTS_DIR"
    "$ROOT/scripts/ci/proxy/deploy.sh" "${server_name}"

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
}

cleanup_proxy_tests() {
    if kubectl get ns proxies; then
        kubectl delete ns proxies --wait
    fi
}

run_proxy_tests() {
    info "Running proxy tests"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: run_proxy_tests <server_name>"
    fi

    local server_name="$1"
    local ping_endpoint="v1/ping"

    info "Test HTTP access to plain HTTP proxy"
    # --retry-connrefused only works when forcing IPv4, see https://github.com/appropriate/docker-curl/issues/5
    local ping_response_http
    ping_response_http="$(
        curl --retry 5 --retry-connrefused -4 --retry-delay 1 --retry-max-time 10 \
        -f \
        http://"${server_name}":10080/"${ping_endpoint}" | jq -r '.status')"
    echo "Got ping response '${ping_response_http}' from '${ping_endpoint}'"
    [[ "${ping_response_http}" == "ok" ]]

    info "Test HTTPS access to multiplexed TLS proxy"
    # --retry-connrefused only works when forcing IPv4, see https://github.com/appropriate/docker-curl/issues/5
    local ping_response_https
    ping_response_https="$(
        curl --cacert "${PROXY_CERTS_DIR}/ca.crt" \
        --retry 5 --retry-connrefused -4 --retry-delay 1 --retry-max-time 10 \
        -f \
        https://"${server_name}":10443/"${ping_endpoint}" | jq -r '.status')"
    echo "Got ping response '${ping_response_https}' from '${ping_endpoint}'"
    [[ "${ping_response_https}" == "ok" ]]

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
        local endpoint="${server_name}:${port}"
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
        roxctl "${extra_args[@]}" --plaintext="$plaintext" -e "${server_name}:${port}" -p "$ROX_PASSWORD" sensor generate k8s --name remote --continue-if-exists || \
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
