#!/usr/bin/env bash

# TLS compliance E2E test.
#
# Verifies that all ACS TLS endpoints use the expected key exchange groups,
# ciphers, and TLS versions after setting a strict OCP TLS profile.
#
# Prerequisites:
#   - OCP 5.0+ cluster with KUBECONFIG set
#   - roxie, roxctl, oc, jq, podman on PATH
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
FAILED=0

# Temporary ML-KEM exceptions. Remove once PG supports ssl_ecdh_curve groups.
# ROX-34978
MLKEM_EXCEPTIONS="central-db:5432|scanner-v4-db:5432"

fail() {
    echo "FAIL: $*" >&2
    FAILED=1
}

# --- Step 1: Build and push tls-scanner image ---
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

# --- Step 2: Enable OCP TLS feature gates and set TLS profile ---

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
                    "X25519MLKEM768",
                    "X25519"
                ]
            }
        }
    }
}'

info "Waiting for API server to stabilize..."
oc wait --for=condition=Available clusteroperator/kube-apiserver --timeout=600s
info "API server stable."

# --- Step 3: Deploy ACS ---

roxie_config="$(mktemp)"
cat > "$roxie_config" <<EOF
roxie:
  version: "${MAIN_IMAGE_TAG}"
  konfluxImages: true
central:
  namespace: ${ACS_NS}
  spec:
    network:
      policies: Disabled
securedCluster:
  namespace: ${ACS_NS}
  spec:
    network:
      policies: Disabled
EOF

info "Deploying ACS with roxie..."
deploy_stackrox_with_roxie "$roxie_config"

# --- Step 4: Run tls-scanner ---

info "Deploying tls-scanner job..."
pushd "$TLS_SCANNER_DIR" > /dev/null

export SCANNER_IMAGE="image-registry.openshift-image-registry.svc:5000/${ACS_NS}/tls-scanner:latest"
export NAMESPACE="$ACS_NS"
export NAMESPACE_FILTER="$ACS_NS"
export ARTIFACT_WAIT="${ARTIFACT_WAIT:-300}"

./deploy.sh deploy

SCAN_RESULTS="$TLS_SCANNER_DIR/artifacts/results.json"

popd > /dev/null

# --- Step 5: Validate results ---

if [[ ! -f "$SCAN_RESULTS" ]]; then
    fail "Scan results not found at $SCAN_RESULTS"
    exit 1
fi

info "Validating scan results..."

# central-db must have TLS enabled (guards against tls-scanner malfunction).
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

# All OK endpoints must support TLS 1.3.
tls13_failures=$(jq -r '
    .ip_results[]? |
    (.openshift_component.component // .pod.Name) as $name |
    .port_results[]? |
    select(.status == "OK") |
    select(.tls13_supported != true) |
    "\($name):\(.port)"
' "$SCAN_RESULTS" | sort -u || true)

if [[ -n "$tls13_failures" ]]; then
    fail "Endpoints without TLS 1.3: ${tls13_failures}"
else
    info "All TLS endpoints support TLS 1.3."
fi

# All OK endpoints must support ML-KEM (with temporary PG exceptions).
# TODO: Once TLS groups adherence is implemented (ROX-34980 / PR #23739),
# tighten this to verify ML-KEM is the *only* offered group.
mlkem_failures=$(jq -r '
    .ip_results[]? |
    (.openshift_component.component // .pod.Name) as $name |
    .port_results[]? |
    select(.status == "OK") |
    select(.mlkem_supported != true) |
    "\($name):\(.port)"
' "$SCAN_RESULTS" | grep -vE "^(${MLKEM_EXCEPTIONS})$" | sort -u || true)

if [[ -n "$mlkem_failures" ]]; then
    fail "Endpoints without ML-KEM: ${mlkem_failures}"
else
    info "All endpoints support ML-KEM (exceptions: ${MLKEM_EXCEPTIONS})."
fi

# --- Step 6: BAT tests ---

info "Running BAT tests..."
roxctl --insecure-skip-tls-verify -e "${API_ENDPOINT}" -p "${ROX_ADMIN_PASSWORD}" \
    central whoami || fail "roxctl central whoami failed"

# --- Step 7: Collect artifacts ---

artifact_dest="${ARTIFACT_DIR:-/tmp/tls-compliance-artifacts}"
mkdir -p "$artifact_dest"
cp -r "$TLS_SCANNER_DIR/artifacts/"* "$artifact_dest/" 2>/dev/null || true
info "Artifacts copied to $artifact_dest"

# Cleanup tls-scanner job.
pushd "$TLS_SCANNER_DIR" > /dev/null
NAMESPACE="$ACS_NS" ./deploy.sh cleanup 2>/dev/null || true
popd > /dev/null

if (( FAILED )); then
    echo "TLS compliance test FAILED."
    exit 1
fi

echo "TLS compliance test PASSED."
