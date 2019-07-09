#!/usr/bin/env bash

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

PLUGIN_VERSION="1.0"

tmpdir="$(mktemp -d)"
gsutil cat "gs://sr-authz-plugin-src/${PLUGIN_VERSION}/default-authz-plugin-${PLUGIN_VERSION}-src.tar.gz" \
	| tar -C "$tmpdir" -xvzf -

plugin_dir="${tmpdir}/default-authz-plugin"

export TLS_CERT_FILE="${DIR}/scopedaccess/config/server-tls.crt"
export TLS_KEY_FILE="${DIR}/scopedaccess/config/server-tls.key"

export SERVER_CONFIG_FILE="${DIR}/scopedaccess/config/server-config.json"
export RULES_FILE="${DIR}/scopedaccess/config/rules.gval"

export AUTHZ_PLUGIN_IMAGE="stackrox/default-authz-plugin:${PLUGIN_VERSION}"

"${plugin_dir}/examples/deployment/deploy.sh"
sleep 5
POD=$(kubectl -n stackrox get pod -o jsonpath='{.items[?(@.metadata.labels.app=="authorization-plugin")].metadata.name}')
echo $POD
kubectl -n stackrox wait --for=condition=ready "pod/$POD" --timeout=2m
