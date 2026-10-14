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
    if ! oc get namespace "${altNamespace}" &>/dev/null; then
        echo "FAIL: neither ${namespace} nor ${altNamespace} namespace exists — ACS not installed" >&2
        exit 1
    fi
    namespace="${altNamespace}"
fi

declare centralStatus
centralStatus=$(oc get central -n "${namespace}" -o jsonpath='{.items[0].status.conditions[?(@.type=="Deployed")].status}' 2>/dev/null || echo "NotFound")
if [[ "${centralStatus}" == "NotFound" ]]; then
    centralStatus=$(oc get deployment central -n "${namespace}" -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "NotFound")
fi

if [[ "${centralStatus}" != "True" ]]; then
    echo "FAIL: ACS Central is not available (status: ${centralStatus})" >&2
    exit 1
fi

echo "PASS: ACS Central is deployed in ${namespace}"
exit 0
true
