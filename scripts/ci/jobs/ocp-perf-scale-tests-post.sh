#!/usr/bin/env bash

# Collect diagnostics after performance/scale testing completes.
# This script collects debug dumps and diagnostics from Central.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
# shellcheck source=../lib.sh
source "$ROOT/scripts/ci/lib.sh"

info "Collecting diagnostics after performance/scale tests"

# Directory names match what PostClusterTest uses in post_tests.py
DEBUG_OUTPUT="debug-dump"
DIAGNOSTIC_OUTPUT="diagnostic-bundle"
CENTRAL_DATA_OUTPUT="central-data"

# Wait for Central API to be responsive
if wait_for_api; then
    info "Central API is responsive, collecting diagnostics"

    # Get central debug dump (includes metrics)
    if get_central_debug_dump "${DEBUG_OUTPUT}"; then
        info "Collected central debug dump to ${DEBUG_OUTPUT}"
        process_central_metrics "${DEBUG_OUTPUT}" || info "Warning: Failed to process central metrics"
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
else
    info "Warning: Central API is not responsive, skipping diagnostic collection"
fi

info "Diagnostic collection complete"
