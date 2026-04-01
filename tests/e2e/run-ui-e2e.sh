#!/usr/bin/env bash

# Runs ui/ e2e tests. Formerly CircleCI gke-ui-e2e-tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"
# shellcheck source=../../scripts/ci/sensor-wait.sh
source "$ROOT/scripts/ci/sensor-wait.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"
# shellcheck source=../../tests/scripts/setup-certs.sh
source "$ROOT/tests/scripts/setup-certs.sh"

set -euo pipefail

enable_console_plugin() {
    info "Enabling advanced-cluster-security console plugin"

    # Wait for the ConsolePlugin CRD and resource to exist
    info "Waiting for ConsolePlugin CRD to be established"
    kubectl wait --for=condition=Established --timeout=120s \
        crd/consoleplugins.console.openshift.io

    info "Waiting for advanced-cluster-security ConsolePlugin resource to be created"
    kubectl wait --for=create --timeout=120s \
        consoleplugin/advanced-cluster-security

    # Enable the plugin in the console operator config (idempotent)
    local current_plugins
    current_plugins=$(oc get console.operator.openshift.io cluster -o jsonpath='{.spec.plugins[*]}' 2>/dev/null || echo "")
    if [[ "$current_plugins" =~ (^|[[:space:]])advanced-cluster-security([[:space:]]|$) ]]; then
        info "Plugin 'advanced-cluster-security' is already enabled"
    else
        info "Enabling plugin 'advanced-cluster-security' in console config"
        if [[ -z "$current_plugins" || "$current_plugins" == "null" ]]; then
            # No plugins enabled yet, initialize the array
            oc patch console.operator.openshift.io cluster --type=merge \
              -p '{"spec":{"plugins":["advanced-cluster-security"]}}'
        else
            # Append to existing plugins
            oc patch console.operator.openshift.io cluster --type=json \
              -p '[{"op": "add", "path": "/spec/plugins/-", "value": "advanced-cluster-security"}]'
        fi

        # Wait for console deployment to roll out with plugin configuration
        info "Waiting for console deployment to roll out with plugin enabled"
        kubectl rollout status deployment/console -n openshift-console --timeout=3m || {
            info "Warning: Console rollout did not complete, proceeding anyway"
        }
    fi
}

test_ui_e2e() {
    info "Starting UI e2e tests"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"

    export DEPLOY_DIR="deploy/${ORCHESTRATOR_FLAVOR}"

    export_test_environment

    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    # deploy the optional components before stackrox
    deploy_optional_e2e_components
    deploy_stackrox

    # Enable the console plugin for OpenShift
    if [[ "${ORCHESTRATOR_FLAVOR}" == "openshift" ]]; then
        enable_console_plugin
    fi

    run_ui_e2e_tests
}

run_ui_e2e_tests() {
    info "Running UI e2e tests"

    if [[ "${LOAD_BALANCER}" == "lb" ]]; then
        local hostname
        if [[ "${API_HOSTNAME}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            info "Getting hostname from IP: ${API_HOSTNAME}"
            hostname=$("$ROOT/tests/e2e/get_hostname.py" "${API_HOSTNAME}")
        else
            hostname="${API_HOSTNAME}"
        fi
        info "Hostname for central-lb alias: ${hostname}"
        echo "central-lb ${hostname}" > /tmp/hostaliases
        export HOSTALIASES=/tmp/hostaliases
        export UI_BASE_URL="https://central-lb:443"
    elif [[ "${LOAD_BALANCER}" == "route" ]]; then
        die "unsupported LOAD_BALANCER ${LOAD_BALANCER}"
    else
        export UI_BASE_URL="https://localhost:${LOCAL_PORT}"
    fi

    make -C ui test-e2e || touch FAIL

    store_test_results "ui/test-results/reports/cypress/integration/." "cy-reps"

    if is_OPENSHIFT_CI; then
        cp -a ui/test-results/artifacts/* "${ARTIFACT_DIR}/" || true
    fi

    [[ ! -f FAIL ]] || die "UI e2e tests failed"
}

test_ui_e2e
