#!/usr/bin/env bash

set -euo pipefail

dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

kubectl apply -f "${dir}/central-plaintext.yaml"

kubectl create ns proxies --dry-run -o yaml | kubectl apply -f -

kubectl label ns proxies --overwrite stackrox-proxies=true

kubectl -n proxies create cm nginx-proxy-plain-http-conf --from-file="${dir}/nginx-proxy-plain-http.conf" \
	--dry-run -o yaml | kubectl apply -f -

kubectl apply -f "${dir}/nginx-proxy-plain-http.yaml"

kubectl -n proxies create cm nginx-proxy-tls-multiplexed-conf \
	--from-file="${dir}/nginx-proxy-tls-multiplexed.conf" \
	--dry-run -o yaml | kubectl apply -f -

cert_dir="${PROXY_CERTS_DIR:-$(mktemp -d)}"
"${dir}/../../../tests/scripts/setup-certs.sh" "${cert_dir}" "central-proxy.stackrox.local" "Proxy CA"

kubectl -n proxies create secret tls nginx-proxy-tls-multiplexed-certs \
	--cert="${cert_dir}/tls.crt" \
	--key="${cert_dir}/tls.key" \
	--dry-run -o yaml | kubectl apply -f -

kubectl apply -f "${dir}/nginx-proxy-tls-multiplexed.yaml"
sleep 5
kubectl -n proxies wait --for=condition=available deploy/nginx-proxy-{plain-http,tls-multiplexed} --timeout=2m
