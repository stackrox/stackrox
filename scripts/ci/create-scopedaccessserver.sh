#!/usr/bin/env bash

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

CMD="$1"

if [[ -z $CMD ]]; then
    >&2 echo "First argument must be command name (kubectl or oc)"
    exit 1
fi

$CMD -n stackrox get secret stackrox >/dev/null 2>&1 || {
	echo >&2 "'stackrox' image pull secrets or namespace do not exist."
	echo >&2 "Please launch StackRox in this cluster before running this script."
	exit 1
}

TLS_CERT_FILE="${TLS_CERT_FILE:-${DIR}/scopedaccess/config/server-tls.crt}"
TLS_KEY_FILE="${TLS_KEY_FILE:-${DIR}/scopedaccess/config/server-tls.key}"

SERVER_CONFIG_FILE="${SERVER_CONFIG_FILE:-${DIR}/scopedaccess/config/server-config.json}"
RULES_FILE="${RULES_FILE:-${DIR}/scopedaccess/config/rules.gval}"

AUTHZ_PLUGIN_IMAGE="${AUTHZ_PLUGIN_IMAGE:-stackrox/default-authz-plugin:latest}"

$CMD -n stackrox create secret tls authz-plugin-tls \
	--cert "${TLS_CERT_FILE}" \
	--key "${TLS_KEY_FILE}" \
	--dry-run -o yaml | kubectl apply -f -

$CMD -n stackrox create configmap authz-plugin-config \
	--from-file server-config.json="${SERVER_CONFIG_FILE}" \
	--from-file rules.gval="${RULES_FILE}" \
	--dry-run -o yaml | kubectl apply -f -

if [[ "${CMD}" == "oc" ]]; then
  $CMD create -f ${DIR}/scopedaccess/deployment/scopedaccess_scc.yaml
fi

sed -e 's@${IMAGE}@'"$AUTHZ_PLUGIN_IMAGE"'@g' <"${DIR}/scopedaccess/deployment/scopedaccess.yaml" | kubectl apply -f -
sleep 5
POD=$($CMD -n stackrox get pod -o jsonpath='{.items[?(@.metadata.labels.app=="authorization-plugin")].metadata.name}')
echo $POD
$CMD  -n stackrox wait --for=condition=ready "pod/$POD" --timeout=2m