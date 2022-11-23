#!/bin/bash

set -e

die() {
  echo >&2 "$@"
  exit 1
}

certs_tmp_dir="$1"
[[ -n "${certs_tmp_dir}" ]] || die "Usage: $0 <certs_dir>"


printstatus() {
    echo current resource status ...
    echo
    kubectl -n stackrox get all -l app=webhookserver
}

trap printstatus ERR

gitroot="$(git rev-parse --show-toplevel)"
[[ -n "${gitroot}" ]] || die "Could not determine git root"

"${gitroot}/tests/scripts/setup-certs.sh" "${certs_tmp_dir}" webhookserver.stackrox "Webhook Server CA"
cd "${gitroot}/webhookserver"
mkdir -p chart/certs
cp "${certs_tmp_dir}/tls.crt" "${certs_tmp_dir}/tls.key" chart/certs
helm -n stackrox upgrade --install webhookserver chart/

sleep 5
pod="$(kubectl -n stackrox get pod -l app=webhookserver -o name)"
echo "Got pod ${pod}"
[[ -n "${pod}" ]]
kubectl -n stackrox wait --for=condition=ready "${pod}" --timeout=5m
nohup kubectl -n stackrox port-forward "${pod}" 8080:8080 </dev/null > /dev/null 2>&1 &
sleep 1
