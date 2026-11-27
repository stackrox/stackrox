#!/bin/bash
set -euxo pipefail
shopt -s inherit_errexit 2>/dev/null || true

if [[ -z "${KUBECONFIG:-}" ]]; then
    echo "KUBECONFIG not set, skipping" >&2
    exit 2
fi

declare namespace="${ACS_NAMESPACE:-stackrox}"

if ! oc get namespace "${namespace}" &>/dev/null; then
    declare altNamespace="rhacs-operator"
    if oc get namespace "${altNamespace}" &>/dev/null; then
        namespace="${altNamespace}"
    else
        echo "FAIL: ACS namespace not found" >&2
        exit 1
    fi
fi

declare scannerReady
scannerReady=$(oc get deployment scanner -n "${namespace}" -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "NotFound")

if [[ "${scannerReady}" == "NotFound" ]]; then
    scannerReady=$(oc get deployment scanner-v4-indexer -n "${namespace}" -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "NotFound")
fi

if [[ "${scannerReady}" != "True" ]]; then
    echo "FAIL: ACS Scanner is not available (status: ${scannerReady})" >&2
    exit 1
fi

echo "PASS: ACS Scanner is deployed and available in ${namespace}"
exit 0
