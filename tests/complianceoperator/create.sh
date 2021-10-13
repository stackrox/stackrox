#! /bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if ! kubectl get crd compliancecheckresults.compliance.openshift.io; then
    kubectl create -R -f "${DIR}/crds"
    kubectl create -R -f "${DIR}/resources"
fi
