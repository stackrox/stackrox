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

declare bundleSecret
bundleSecret=$(oc get secrets -n "${namespace}" --no-headers 2>/dev/null | grep -E "init-bundle|collector-tls|sensor-tls|admission-control-tls" | head -1 | awk '{print $1}')
if [[ -z "${bundleSecret}" ]]; then
    echo "FAIL: no ACS init-bundle or TLS secrets found in ${namespace}" >&2
    exit 1
fi

declare -i tlsSecretCount=0
for secretName in collector-tls sensor-tls admission-control-tls; do
    if oc get secret "${secretName}" -n "${namespace}" &>/dev/null; then
        tlsSecretCount+=1
    fi
done

if [[ ${tlsSecretCount} -lt 3 ]]; then
    echo "WARN: only ${tlsSecretCount}/3 ACS TLS secrets found (collector-tls, sensor-tls, admission-control-tls)" >&2
fi

echo "PASS: ACS init-bundle configured — ${tlsSecretCount}/3 TLS secrets present in ${namespace}"
exit 0
