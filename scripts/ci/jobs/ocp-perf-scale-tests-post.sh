#!/usr/bin/env bash

# Collect diagnostics after performance/scale testing completes.
# This script collects debug dumps and diagnostics from Central.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
# shellcheck source=../lib.sh
source "$ROOT/scripts/ci/lib.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"

info "Collecting diagnostics after performance/scale tests"

# Directory names match what PostClusterTest uses in post_tests.py
DEBUG_OUTPUT="debug-dump"
DIAGNOSTIC_OUTPUT="diagnostic-bundle"
CENTRAL_DATA_OUTPUT="central-data"

# Set up port forward to Central with retry
info "Setting up port forward to central"
PORT_FORWARD_PID=""
MAX_PORT_FORWARD_RETRIES=5
for retry in $(seq 1 ${MAX_PORT_FORWARD_RETRIES}); do
    info "Attempting to establish port-forward (try ${retry}/${MAX_PORT_FORWARD_RETRIES})"

    # Kill any existing port-forward on this port
    pkill -f "port-forward.*8000:443" || true
    sleep 1

    # Start new port-forward
    nohup kubectl -n stackrox port-forward svc/central 8000:443 >/dev/null 2>&1 &
    PORT_FORWARD_PID=$!

    # Wait for port-forward to be ready
    for i in $(seq 1 10); do
        if curl -sk --connect-timeout 2 --max-time 5 https://localhost:8000/v1/ping >/dev/null 2>&1; then
            info "Port-forward established successfully"
            break 2  # Break out of both loops
        fi
        sleep 1
    done

    # If we got here, port-forward failed
    info "Port-forward attempt ${retry} failed, retrying..."
    kill "${PORT_FORWARD_PID}" 2>/dev/null || true
done

if ! curl -sk --connect-timeout 2 --max-time 5 https://localhost:8000/v1/ping >/dev/null 2>&1; then
    info "Error: Failed to establish port-forward after ${MAX_PORT_FORWARD_RETRIES} attempts"
    exit 1
fi

# Set API_ENDPOINT since we know it from the port-forward
export API_ENDPOINT="localhost:8000"

# Get admin password from SHARED_DIR (saved by stackrox-install-helm step)
if [[ -n "${SHARED_DIR:-}" && -f "${SHARED_DIR}/rox_admin_password" ]]; then
    export ROX_ADMIN_PASSWORD="$(cat "${SHARED_DIR}/rox_admin_password")"
    info "Retrieved admin password from SHARED_DIR"
else
    info "Warning: Could not find admin password in SHARED_DIR"
fi

# Wait for Central API to be responsive (ignore ci_export permission errors at end of function)
# If wait_for_api truly fails (API down), it will exit the script with exit 1
# If it succeeds but ci_export fails, || true catches that and we continue
wait_for_api || true
info "Central API is responsive, collecting diagnostics"

# Get central debug dump (includes metrics)
if get_central_debug_dump "${DEBUG_OUTPUT}"; then
    info "Collected central debug dump to ${DEBUG_OUTPUT}"
    # Skip metrics processing - debug dump already contains raw metrics in the zip
else
    info "Warning: Failed to collect central debug dump"
fi

# Get central diagnostics bundle
if get_central_diagnostics "${DIAGNOSTIC_OUTPUT}"; then
    info "Collected central diagnostics to ${DIAGNOSTIC_OUTPUT}"
else
    info "Warning: Failed to collect central diagnostics"
fi

# Grab additional data from Central
if "$ROOT/scripts/grab-data-from-central.sh" "${CENTRAL_DATA_OUTPUT}"; then
    info "Collected central data to ${CENTRAL_DATA_OUTPUT}"
else
    info "Warning: Failed to collect central data"
fi

# Store artifacts to OpenShift CI artifact directory
if [[ -n "${ARTIFACT_DIR:-}" ]]; then
    info "Copying diagnostics to ${ARTIFACT_DIR}"
    for dir in "${DEBUG_OUTPUT}" "${DIAGNOSTIC_OUTPUT}" "${CENTRAL_DATA_OUTPUT}"; do
        if [[ -d "${dir}" ]]; then
            cp -r "${dir}" "${ARTIFACT_DIR}/" || info "Warning: Failed to copy ${dir}"
        fi
    done
else
    info "Warning: ARTIFACT_DIR not set, diagnostics not copied to artifacts"
fi

# Clean up port forward
if [[ -n "${PORT_FORWARD_PID:-}" ]]; then
    kill "${PORT_FORWARD_PID}" 2>/dev/null || true
fi

info "Diagnostic collection complete"
