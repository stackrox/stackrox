#! /bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if ! kubectl get crd compliancecheckresults.compliance.openshift.io; then
    kubectl apply -R -f "${DIR}/crds"

    # Wait for all compliance CRDs to be established before creating resources
    # that depend on them. Without this, applying custom resources could fail.
    # awk command will provide list of CRDs from "crds" dir.
    # Format example: "crd/compliancecheckresults.compliance.openshift.io"
    all_crds=()
    while IFS='' read -r single_crd; do all_crds+=("$single_crd"); done < <(grep -rh '^  name:.*\.compliance\.openshift\.io' "${DIR}/crds/" | awk '{print "crd/" $2}')
    kubectl wait --for condition=established --timeout=60s "${all_crds[@]}"

    # `kubectl wait --for condition=established` can fail immediately
    # (not time out) when a just-applied CRD has not had its
    # .status.conditions populated by the API server yet.
    established=false
    for _ in $(seq 1 5); do
        if kubectl wait --for condition=established --timeout=60s "${all_crds[@]}"; then
            established=true
            break
        fi
        echo "kubectl wait for established compliance CRDs failed (status may not be populated yet); retrying in 5s..."
        sleep 5
    done
    if [ "${established}" != "true" ]; then
        echo "ERROR: compliance CRDs did not become established in time" >&2
        exit 1
    fi

    kubectl apply -R -f "${DIR}/resources"
fi
