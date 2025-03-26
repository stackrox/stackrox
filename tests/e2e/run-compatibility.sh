#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Runs all e2e go tests marked as compatibility.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

# shellcheck source=../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"
# shellcheck source=../../scripts/ci/sensor-wait.sh
source "$ROOT/scripts/ci/sensor-wait.sh"
# shellcheck source=../../tests/scripts/setup-certs.sh
source "$ROOT/tests/scripts/setup-certs.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"

run_compatibility_tests() {
    if [[ "$#" -ne 2 ]]; then
        info "run_compatibility_tests() Args: $*, Numargs: $#"
        die "missing args. usage: run_compatibility_tests <central_version> <sensor_version>"
    fi

    local central_version="$1"
    local sensor_version="$2"

    short_central_tag="$(shorten_tag "${central_version}")"
    short_sensor_tag="$(shorten_tag "${sensor_version}")"

    info "Starting go compatibility tests with central v${short_central_tag} and sensor v${short_sensor_tag}"

    _run_compatibility_tests "${central_version}" "${sensor_version}" "${short_central_tag}" "${short_sensor_tag}"
}

_run_compatibility_tests() {
    local central_version="$1"
    local sensor_version="$2"
    local short_central_tag="$3"
    local short_sensor_tag="$4"
    local compatibility_dir="compatibility-test-central-v${short_central_tag}-sensor-v${short_sensor_tag}"

    info "Starting test (go compatibility test Central v${short_central_tag}, Sensor v${short_sensor_tag})"

    require_environment "KUBECONFIG"

    export_test_environment
    ci_export CENTRAL_PERSISTENCE_NONE "true"

    export SENSOR_HELM_DEPLOY=true
    export ROX_ACTIVE_VULN_REFRESH_INTERVAL=1m
    export ROX_NETPOL_FIELDS=true
    export ROX_SENSOR_UPGRADER_ENABLED=false

    test_preamble
    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs
    info "Creating mocked compliance operator data for compliance v1 tests"
    "$ROOT/tests/complianceoperator/create.sh"

    # For every version pair we use a unique name for the secured cluster to prevent attempts
    # to register the same secured cluster name multiple times, which will fails in CRS mode.
    export CLUSTER="sc-${short_sensor_tag}-${short_central_tag}"
    deploy_stackrox_with_custom_central_and_sensor_versions "${central_version}" "${sensor_version}"
    echo "Stackrox deployed with Central version - ${central_version}, Sensor version - ${sensor_version}"

    rm -f FAIL

    prepare_for_endpoints_test

    # Give some time for stackrox to be reachable
    wait_for_api

    info "API version compatibility tests"
    if pr_has_label "ci-release-build"; then
        echo "Running version compatibility tests in release mode"
        export GOTAGS=release
    fi
    kubectl -n stackrox get pods
    make -C tests compatibility-tests || touch FAIL

    update_junit_prefix_with_central_and_sensor_version "${short_central_tag}" "${short_sensor_tag}" "${ROOT}/tests/compatibility-tests-results"

    store_test_results "tests/compatibility-tests-results" "${compatibility_dir}"
    [[ ! -f FAIL ]] || die "compatibility tests failed for Central v${short_central_tag}, Sensor v${short_sensor_tag}"

    cd "$ROOT"

    collect_and_check_stackrox_logs "/tmp/compatibility-test-logs" "${compatibility_dir}/initial_tests"
}

# Duplicate function with run.sh
# TODO(ROX-26592): Remove duplication
test_preamble() {
    require_executable "roxctl"

    export ROX_PLAINTEXT_ENDPOINTS="8080,grpc@8081"
    export ROXDEPLOY_CONFIG_FILE_MAP="$ROOT/scripts/ci/endpoints/endpoints.yaml"
    export TRUSTED_CA_FILE="$ROOT/tests/bad-ca/root.crt"
}

# Duplicate function with run.sh
# TODO(ROX-26592): Remove duplication
prepare_for_endpoints_test() {
    info "Preparation for endpoints_test.go"

    local gencerts_dir
    gencerts_dir="$(mktemp -d)"
    setup_client_CA_auth_provider
    setup_generated_certs_for_test "$gencerts_dir"
    if [[ ${ORCHESTRATOR_FLAVOR:-} == "openshift" ]]; then
        info "Skipping resource patching for skipped endpoints_test.go. TODO(ROX-24688)"
    else
        patch_resources_for_test
    fi
    export SERVICE_CA_FILE="$gencerts_dir/ca.pem"
    export SERVICE_CERT_FILE="$gencerts_dir/sensor-cert.pem"
    export SERVICE_KEY_FILE="$gencerts_dir/sensor-key.pem"
}

shorten_tag() {
    if [[ "$#" -ne 1 ]]; then
        die "Expected a version tag as parameter in shorten_tag: shorten_tag <tag>"
    fi

    input_tag="$1"

    development_version_regex='([0-9]+\.[0-9]+\.[xX])'
    with_minor_version_regex='([0-9]+\.[0-9]+)\.[0-9]+'

    if [[ $input_tag =~ $development_version_regex ]]; then
        echo "${BASH_REMATCH[1]}"
    elif [[ $input_tag =~ $with_minor_version_regex ]]; then
        echo "${BASH_REMATCH[1]}.z"
    else
        echo "${input_tag}"
        >&2 echo "Failed to shorten tag ${input_tag} as it did not match any of the following regexes:
        \"${development_version_regex}\", \"${with_minor_version_regex}\""
        exit 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -ne 2 ]]; then
        info "run-compatibility.sh Args: $*, Numargs: $#"
        die "missing args. usage: run-compatibility.sh <central_version> <sensor_version>"
    fi
    run_compatibility_tests "$1" "$2"
fi
