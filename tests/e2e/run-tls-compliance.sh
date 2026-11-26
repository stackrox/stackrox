#!/usr/bin/env bash

# TLS compliance E2E test.
#
# Verifies that all ACS and RHACS operator TLS endpoints use the expected key
# exchange groups and TLS versions after setting a strict OCP TLS profile.
#
# Prerequisites:
#   - OCP 5.0+ cluster with custom TLS profile support and KUBECONFIG set
#   - roxie, oc, jq, podman on PATH
#   - tls-scanner checkout at $TLS_SCANNER_DIR
#
# With no arguments, build and push the scanner before running the test.
# CI runs build-scanner, push-scanner, and test separately so the build
# can run while the cluster is provisioning.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "${ROOT}/scripts/ci/lib.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "${ROOT}/tests/e2e/lib.sh"

TLS_SCANNER_DIR="${TLS_SCANNER_DIR:?Set TLS_SCANNER_DIR to the openshift/tls-scanner checkout}"
TLS_SCANNER_LOCAL_IMAGE="localhost/tls-scanner:latest"
SCAN_RESULTS=""
ACS_NS="stackrox"
# Namespace used by roxie when Central and SecuredCluster share the same version.
OPERATOR_NS="rhacs-operator-system"
FAILED=0

# TODO(ROX-34978): Temporary exceptions from the TLS key exchange groups check.
# Remove once PostgreSQL honors the TLS profile's group list.
TLS_GROUPS_EXCEPTIONS="central-db:5432|scanner-v4-db:5432"

fail() {
    echo "FAIL: $*" >&2
    FAILED=1
}

build_tls_scanner() (
    cd "$TLS_SCANNER_DIR"
    # deploy.sh otherwise queries the current OpenShift project, even for builds.
    NAMESPACE="$ACS_NS" SCANNER_IMAGE="$TLS_SCANNER_LOCAL_IMAGE" \
        DOCKERFILE=Dockerfile.local TARGET_PLATFORM=linux/amd64 ./deploy.sh build
)

push_tls_scanner() {
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
    podman login "$registry" --username builder --password-stdin --tls-verify=false <<< "$token"

    info "Pushing tls-scanner..."
    podman push --tls-verify=false "$TLS_SCANNER_LOCAL_IMAGE" "docker://${registry}/${ACS_NS}/tls-scanner:latest"
}

set_tls_profile() {
    local master_config_before worker_config_before
    local master_config_after worker_config_after
    local i

    master_config_before=$(oc get mcp master -o jsonpath='{.status.configuration.name}')
    worker_config_before=$(oc get mcp worker -o jsonpath='{.status.configuration.name}')
    info "MCP configs before changes: master=$master_config_before worker=$worker_config_before"

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
  crsIssueRetries: 60
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
    local db_status operator_ok_endpoints tls_version_failures tls_groups_failures mlkem_supported_failures

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

    # tls-scanner sanity check: the DBs' port 5432 must be OK, not NO_TLS.
    # (tls-scanner may report TLS disabled due to the STARTTLS protocol used by Postgres)
    local db
    for db in central-db scanner-v4-db; do
        db_status=$(jq -r --arg prefix "${db}-" '
            [.ip_results[]? |
             select(.pod.Name | startswith($prefix)) |
             .port_results[]? |
             select(.port == 5432) |
             .status] | first // "MISSING"
        ' "$SCAN_RESULTS")

        if [[ "$db_status" != "OK" ]]; then
            fail "${db}:5432 status is '${db_status}', expected 'OK'"
        else
            info "${db}:5432 TLS OK."
        fi
    done

    # Make sure we get at least one TLS scan result for at least one operator pod.
    operator_ok_endpoints=$(jq -r --arg ns "$OPERATOR_NS" '
        [.ip_results[]? |
         select(.pod.Namespace == $ns) |
         .port_results[]? |
         select(.status == "OK")] | length
    ' "$SCAN_RESULTS")

    if [[ "$operator_ok_endpoints" -eq 0 ]]; then
        fail "No OK TLS scan result for any operator pod in ${OPERATOR_NS}"
    else
        info "Operator TLS endpoints OK (${operator_ok_endpoints} in ${OPERATOR_NS})."
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
    ' "$SCAN_RESULTS" | sort -u)

    if [[ -n "$tls_version_failures" ]]; then
        fail "Endpoints not offering only TLS 1.3: ${tls_version_failures}"
    else
        info "All TLS endpoints offer only TLS 1.3."
    fi

    # OK endpoints must offer only X25519MLKEM768 (profile groups list).
    # The DBs currently offer X25519MLKEM768:prime256v1 until they honor the TLS profile.
    tls_groups_failures=$(jq -r '
        .ip_results[]? |
        (.openshift_component.component // .pod.Name) as $name |
        .port_results[]? |
        select(.status == "OK") |
        select(
            (.tls_key_exchange.groups | type) != "array" or
            .tls_key_exchange.groups != ["X25519MLKEM768"]
        ) |
        "\($name):\(.port) groups=\(.tls_key_exchange.groups // [] | join(","))"
    ' "$SCAN_RESULTS" | grep -vE "^(${TLS_GROUPS_EXCEPTIONS}) " | sort -u || true)

    if [[ -n "$tls_groups_failures" ]]; then
        fail "Endpoints not offering only X25519MLKEM768: ${tls_groups_failures}"
    else
        info "All endpoints offer only X25519MLKEM768 (exceptions: ${TLS_GROUPS_EXCEPTIONS})."
    fi

    # All OK endpoints must support ML-KEM, including the excepted ones.
    mlkem_supported_failures=$(jq -r '
        .ip_results[]? |
        (.openshift_component.component // .pod.Name) as $name |
        .port_results[]? |
        select(.status == "OK") |
        select(.mlkem_supported != true) |
        "\($name):\(.port)"
    ' "$SCAN_RESULTS" | sort -u)

    if [[ -n "$mlkem_supported_failures" ]]; then
        fail "Endpoints without ML-KEM: ${mlkem_supported_failures}"
    else
        info "All endpoints support ML-KEM."
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

case "${1:-all}" in
    build-scanner)
        build_tls_scanner
        exit 0
        ;;
    push-scanner)
        push_tls_scanner
        exit 0
        ;;
    all)
        build_tls_scanner
        push_tls_scanner
        ;;
    test) ;;
    *)
        echo "Usage: $0 [build-scanner|push-scanner|test]" >&2
        exit 1
        ;;
esac

set_tls_profile
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
