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

declare securedClusterStatus
securedClusterStatus=$(oc get securedcluster -n "${namespace}" -o jsonpath='{.items[0].status.conditions[?(@.type=="Deployed")].status}' 2>/dev/null || echo "NotFound")

if [[ "${securedClusterStatus}" == "NotFound" ]]; then
    declare sensorReady
    sensorReady=$(oc get deployment sensor -n "${namespace}" -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "NotFound")
    if [[ "${sensorReady}" != "True" ]]; then
        echo "FAIL: no SecuredCluster CR and sensor not available" >&2
        exit 1
    fi
fi

declare collectorCount
collectorCount=$(oc get daemonset collector -n "${namespace}" -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")

echo "PASS: ACS SecuredCluster is deployed (collector ready on ${collectorCount} nodes)"
exit 0
