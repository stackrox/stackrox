#!/usr/bin/env bash

# TLS compliance E2E test.
#
# Verifies that all ACS and RHACS operator TLS endpoints use the expected key
# exchange groups and TLS versions after setting a strict OCP TLS profile.
#
# Prerequisites:
#   - OCP 5.0+ cluster with KUBECONFIG set
#   - roxie, oc, jq, podman on PATH
#   - tls-scanner checkout at $TLS_SCANNER_DIR

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "${ROOT}/scripts/ci/lib.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "${ROOT}/tests/e2e/lib.sh"

TLS_SCANNER_DIR="${TLS_SCANNER_DIR:?Set TLS_SCANNER_DIR to the openshift/tls-scanner checkout}"
SCAN_RESULTS=""
ACS_NS="stackrox"
# Namespace used by roxie when Central and SecuredCluster share the same version.
OPERATOR_NS="rhacs-operator-system"
FAILED=0

# TODO(ROX-34978): Temporary ML-KEM exceptions. Remove once PG supports ssl_ecdh_curve groups.
MLKEM_EXCEPTIONS="central-db:5432|scanner-v4-db:5432"

fail() {
    echo "FAIL: $*" >&2
    FAILED=1
}

build_and_push_tls_scanner() {
    local registry token

    info "Setting up image registry..."
    oc patch configs.imageregistry.operator.openshift.io/cluster --type merge \
        -p '{"spec":{"defaultRoute":true}}'

    info "Waiting for image registry route..."
    wait_for_object_to_appear "openshift-image-registry" "route/default-route" 300

    registry=$(oc get route default-route -n openshift-image-registry -o jsonpath='{.spec.host}')
    oc create namespace "$ACS_NS" 2>/dev/null || true
    wait_for_object_to_appear "$ACS_NS" "serviceaccount/builder" 60
    token=$(oc create token builder -n "$ACS_NS" --duration=1h)

    info "Logging in to image registry at $registry..."
    podman login "$registry" --username builder --password "$token" --tls-verify=false

    pushd "$TLS_SCANNER_DIR" > /dev/null

    export SCANNER_IMAGE="${registry}/${ACS_NS}/tls-scanner:latest"
    export DOCKERFILE="Dockerfile.local"
    export BUILD_PLATFORMS="linux/amd64"

    info "Building and pushing tls-scanner..."
    ./deploy.sh build
    ./deploy.sh push

    popd > /dev/null
}

enable_tls_feature_gates_and_set_profile() {
    local master_config_before worker_config_before
    local master_config_after worker_config_after
    local i

    master_config_before=$(oc get mcp master -o jsonpath='{.status.configuration.name}')
    worker_config_before=$(oc get mcp worker -o jsonpath='{.status.configuration.name}')
    info "MCP configs before changes: master=$master_config_before worker=$worker_config_before"

    info "Enabling TLSGroupPreferences and TLSAdherence feature gates..."
    oc patch featuregate cluster --type=merge -p '{
        "spec": {
            "featureSet": "CustomNoUpgrade",
            "customNoUpgrade": {
                "enabled": ["TLSGroupPreferences", "TLSAdherence"]
            }
        }
    }'

    info "Waiting for feature gate CRD fields..."
    for i in $(seq 1 12); do
        if oc explain apiserver.spec.tlsSecurityProfile.custom.groups &>/dev/null; then
            info "Feature gates active."
            break
        fi
        if [[ "$i" -eq 12 ]]; then
            fail "CRD fields did not appear within 120s"
            exit 1
        fi
        sleep 10
    done

    info "Setting TLS profile..."
    oc patch apiserver cluster --type=merge -p '{
        "spec": {
            "tlsAdherence": "StrictAllComponents",
            "tlsSecurityProfile": {
                "type": "Custom",
                "custom": {
                    "minTLSVersion": "VersionTLS13",
                    "groups": [
                        "X25519MLKEM768"
                    ]
                }
            }
        }
    }'

    info "Waiting for MCO to pick up TLS profile change..."
    for i in $(seq 1 30); do
        master_config_after=$(oc get mcp master -o jsonpath='{.spec.configuration.name}')
        worker_config_after=$(oc get mcp worker -o jsonpath='{.spec.configuration.name}')
        if [[ "$master_config_after" != "$master_config_before" || "$worker_config_after" != "$worker_config_before" ]]; then
            info "MCO picked up TLS change: master=$master_config_after worker=$worker_config_after"
            break
        fi
        if [[ "$i" -eq 30 ]]; then
            info "MCO did not pick up TLS change within 300s, proceeding anyway."
            break
        fi
        sleep 10
    done

    info "Waiting for MCO to finish rolling out to all nodes (this can take 30+ minutes)..."
    oc get mcp
    oc get nodes
    oc wait mcp --all --for=condition=Updated --timeout=3600s
    info "MCO rollout complete."
    oc get mcp
    oc get nodes
}

deploy_acs() {
    local roxie_config
    roxie_config="$(mktemp)"
    cat > "$roxie_config" <<EOF
roxie:
  version: "${MAIN_IMAGE_TAG}"
  konfluxImages: true
central:
  namespace: ${ACS_NS}
  resourceProfile: ci
  spec:
    network:
      policies: Disabled
securedCluster:
  namespace: ${ACS_NS}
  resourceProfile: ci
  spec:
    network:
      policies: Disabled
    perNode:
      fileActivityMonitoring:
        mode: Enabled
EOF

    info "Deploying ACS with roxie..."
    deploy_stackrox_with_roxie "$roxie_config"
}

run_tls_scanner() {
    info "Deploying tls-scanner job..."
    pushd "$TLS_SCANNER_DIR" > /dev/null

    export SCANNER_IMAGE="image-registry.openshift-image-registry.svc:5000/${ACS_NS}/tls-scanner:latest"
    export NAMESPACE="$ACS_NS"
    export NAMESPACE_FILTER="${ACS_NS},${OPERATOR_NS}"
    export ARTIFACT_WAIT="${ARTIFACT_WAIT:-300}"

    ./deploy.sh deploy

    SCAN_RESULTS="$TLS_SCANNER_DIR/artifacts/results.json"

    popd > /dev/null
}

check_pod_logs_for_tls_errors() {
    local ns
    for ns in "$ACS_NS" "$OPERATOR_NS"; do
        check_pod_logs_for_tls_errors_in_ns "$ns"
    done
}

check_pod_logs_for_tls_errors_in_ns() {
    local ns="$1"
    local TLS_ERROR_PATTERN pod pod_errors tls_errors pod_list pods logs
    local log_retrieval_failed=0

    TLS_ERROR_PATTERN='TLS handshake error|tls: handshake failure|tls: protocol version|tls: no cipher suite|tls: client offered only unsupported|SSL_ERROR|SSL routines|Handshake failed'
    info "Checking pod logs for TLS errors in ${ns}..."
    tls_errors=""

    if ! pod_list=$(oc get pods -n "$ns" -o name); then
        fail "Failed to list pods in namespace ${ns}"
        return
    fi

    pods=$(printf '%s\n' "$pod_list" | grep -v tls-scanner || true)
    if [[ -z "$pods" ]]; then
        fail "No pods found in namespace ${ns} to check for TLS errors"
        return
    fi

    while IFS= read -r pod; do
        [[ -z "$pod" ]] && continue
        if ! logs=$(oc logs "$pod" -n "$ns" --all-containers 2>&1); then
            fail "Failed to get logs for ${ns}/${pod}"
            info "$logs"
            log_retrieval_failed=1
            continue
        fi
        pod_errors=$(printf '%s\n' "$logs" | grep -iE "$TLS_ERROR_PATTERN" || true)
        if [[ -n "$pod_errors" ]]; then
            info "TLS errors in ${ns}/${pod}:"
            echo "$pod_errors"
            tls_errors+="$pod_errors"$'\n'
        fi
    done <<< "$pods"

    if [[ -n "$tls_errors" ]]; then
        fail "TLS errors found in pod logs in ${ns}"
    elif (( log_retrieval_failed )); then
        info "Skipped TLS error summary for ${ns} because some pod logs could not be retrieved."
    else
        info "No TLS errors in pod logs in ${ns}."
    fi
}

validate_scan_results() {
    local central_db_status operator_metrics_status tls_version_failures mlkem_failures

    if [[ ! -f "$SCAN_RESULTS" ]]; then
        fail "Scan results not found at $SCAN_RESULTS"
        exit 1
    fi

    info "Validating scan results..."

    info "Scan summary per endpoint:"
    jq -r '
        .ip_results[]? |
        (.openshift_component.component // .pod.Name) as $name |
        .port_results[]? |
        select(.status == "OK") |
        "\($name):\(.port) tls_versions=\(.tls_versions // [] | join(",")) ciphers=\(.tls_ciphers // [] | join(",")) groups=\(.tls_key_exchange.groups // [] | join(",")) mlkem=\(.mlkem_supported)"
    ' "$SCAN_RESULTS" | sort -u

    # tls-scanner sanity check: central-db:5432 must be OK (STARTTLS probe worked).
    # (tls-scanner may report TLS disabled due to the STARTTLS protocol used by Postgres)
    central_db_status=$(jq -r '
        [.ip_results[]? |
         select(.pod.Name | startswith("central-db-")) |
         .port_results[]? |
         select(.port == 5432) |
         .status] | first // "MISSING"
    ' "$SCAN_RESULTS")

    if [[ "$central_db_status" != "OK" ]]; then
        fail "central-db:5432 status is '${central_db_status}', expected 'OK'"
    else
        info "central-db:5432 TLS OK."
    fi

    # The operator's only TLS endpoint is the metrics server; ensure it was actually scanned.
    operator_metrics_status=$(jq -r --arg ns "$OPERATOR_NS" '
        [.ip_results[]? |
         select(.pod.Namespace == $ns) |
         .port_results[]? |
         select(.port == 8443) |
         .status] | first // "MISSING"
    ' "$SCAN_RESULTS")

    if [[ "$operator_metrics_status" != "OK" ]]; then
        fail "operator metrics ${OPERATOR_NS}:8443 status is '${operator_metrics_status}', expected 'OK'"
    else
        info "operator metrics ${OPERATOR_NS}:8443 TLS OK."
    fi

    # All OK endpoints must offer only TLS 1.3 (profile sets minTLSVersion: VersionTLS13).
    tls_version_failures=$(jq -r '
        .ip_results[]? |
        (.openshift_component.component // .pod.Name) as $name |
        .port_results[]? |
        select(.status == "OK") |
        select(
            (.tls_versions | type) != "array" or
            (.tls_versions | length) == 0 or
            any(.tls_versions[]; . != "TLSv1.3")
        ) |
        "\($name):\(.port) versions=\(.tls_versions // [] | join(","))"
    ' "$SCAN_RESULTS" | sort -u || true)

    if [[ -n "$tls_version_failures" ]]; then
        fail "Endpoints not offering only TLS 1.3: ${tls_version_failures}"
    else
        info "All TLS endpoints offer only TLS 1.3."
    fi

    # OK endpoints must offer only X25519MLKEM768 (profile groups list).
    mlkem_failures=$(jq -r '
        .ip_results[]? |
        (.openshift_component.component // .pod.Name) as $name |
        .port_results[]? |
        select(.status == "OK") |
        select(
            (.tls_key_exchange.groups | type) != "array" or
            .tls_key_exchange.groups != ["X25519MLKEM768"]
        ) |
        "\($name):\(.port) groups=\(.tls_key_exchange.groups // [] | join(","))"
    ' "$SCAN_RESULTS" | grep -vE "^(${MLKEM_EXCEPTIONS}) " | sort -u || true)

    if [[ -n "$mlkem_failures" ]]; then
        fail "Endpoints not offering only X25519MLKEM768: ${mlkem_failures}"
    else
        info "All endpoints offer only X25519MLKEM768 (exceptions: ${MLKEM_EXCEPTIONS})."
    fi
}

collect_artifacts() {
    local artifact_dest

    artifact_dest="${ARTIFACT_DIR:-/tmp/tls-compliance-artifacts}"
    mkdir -p "$artifact_dest"
    cp -r "$TLS_SCANNER_DIR/artifacts/"* "$artifact_dest/" 2>/dev/null || true
    info "Artifacts copied to $artifact_dest"
}

cleanup_tls_scanner() {
    if pushd "$TLS_SCANNER_DIR" > /dev/null; then
        NAMESPACE="$ACS_NS" ./deploy.sh cleanup 2>/dev/null || true
        popd > /dev/null || true
    fi
}

# validate_scan_results exits before the normal collection call when results are missing.
on_exit() {
    local exit_status=$?
    local original_dir

    set +e
    original_dir="$(pwd)"
    collect_artifacts
    cleanup_tls_scanner
    cd "$original_dir"
    exit "$exit_status"
}

build_and_push_tls_scanner
enable_tls_feature_gates_and_set_profile
deploy_acs
trap on_exit EXIT
check_pod_logs_for_tls_errors
run_tls_scanner
validate_scan_results

if (( FAILED )); then
    echo "TLS compliance test FAILED."
    exit 1
fi

echo "TLS compliance test PASSED."
