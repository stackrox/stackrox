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

    kubectl apply -R -f "${DIR}/resources"
fi
