#!/bin/bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/lib.sh"

set -e

certs_tmp_dir="$1"
[[ -n "${certs_tmp_dir}" ]] || die "Usage: $0 <certs_dir>"

printstatus() {
    echo current resource status ...
    echo
    kubectl -n stackrox get all -l app=webhookserver
}

trap printstatus ERR

"${ROOT}/tests/scripts/setup-certs.sh" "${certs_tmp_dir}" webhookserver.stackrox "Webhook Server CA"
cd "${ROOT}/webhookserver"
mkdir -p chart/certs
cp "${certs_tmp_dir}/tls.crt" "${certs_tmp_dir}/tls.key" chart/certs
helm -n stackrox upgrade --install webhookserver chart/

sleep 5
pod="$(kubectl -n stackrox get pod -l app=webhookserver -o name)"
echo "Got pod ${pod}"
[[ -n "${pod}" ]]
kubectl -n stackrox wait --for=condition=ready "${pod}" --timeout=5m

echo "Testing that 8080 is available for port-forward"
exitstatus=0
timeout 5s kubectl -n stackrox port-forward "${pod}" 8080:8080 || exitstatus="$?"
if [[ "${exitstatus}" != "124" ]]; then
    die "ERROR: local port 8080 is not available for webhookserver port-forward"
fi

echo "Starting background 8080 port-forward for webhookserver"
nohup kubectl -n stackrox port-forward "${pod}" 8080:8080 </dev/null > /dev/null 2>&1 &
sleep 1
