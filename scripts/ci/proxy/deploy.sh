#!/usr/bin/env bash

set -euo pipefail

dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

server_name="${1:-central-proxy.stackrox.local}"

kubectl apply -f "${dir}/central-plaintext.yaml"

kubectl create ns proxies --dry-run=client -o yaml | kubectl apply -f -

kubectl label ns proxies --overwrite stackrox-proxies=true

kubectl -n proxies create cm nginx-proxy-plain-http-conf \
	--from-file=nginx-proxy-plain-http.conf=<(server_name="${server_name}" envsubst <"${dir}/nginx-proxy-plain-http.conf") \
	--dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f "${dir}/nginx-proxy-plain-http.yaml"

cert_dir="${PROXY_CERTS_DIR:-$(mktemp -d)}"

echo "Cert directory: ${cert_dir}"
"${dir}/../../../tests/scripts/setup-certs.sh" "${cert_dir}" "${server_name}" "Proxy CA"

kubectl -n proxies create secret tls nginx-proxy-tls-certs \
	--cert="${cert_dir}/tls.crt" \
	--key="${cert_dir}/tls.key" \
	--dry-run=client -o yaml | kubectl apply -f -

for proxy_type in http1 http1-plain http2 http2-plain multiplexed multiplexed-tls-be grpc; do
	proxy_name="nginx-proxy-tls-${proxy_type}"

	kubectl -n proxies create cm "${proxy_name}-conf" \
		--from-file="${proxy_name}.conf"=<(server_name="${server_name}" envsubst <"${dir}/${proxy_name}.conf") \
		--dry-run=client -o yaml | kubectl apply -f -

	kubectl apply -n proxies -f <(name="${proxy_name}" envsubst <"${dir}/nginx-proxy-tls.yaml.template")
done

sleep 5
kubectl -n proxies wait --for=condition=available \
	deploy/nginx-proxy-{plain-http,tls-multiplexed,tls-http1,tls-http1-plain,tls-http2,tls-http2-plain,tls-multiplexed-tls-be} \
	--timeout=2m
