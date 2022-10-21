#!/usr/bin/env bash

set -ueo pipefail

C_YELLOW='\033[0;33m'
C_PURPLE='\033[0;35m'
C_OFF='\033[0m'

if [[ $# -eq 0 ]]; then
    echo -e "${C_YELLOW}Usage: $0 <CLUSTER_NAME>${C_OFF}"
    exit 1
fi

CLUSTER_NAME=$1
ARTIFACTS_DIR="$(mktemp -d)/artifacts"
export KUBECONFIG="${ARTIFACTS_DIR}/kubeconfig"

function download_cluster_artifacts() {
    mkdir -p "${ARTIFACTS_DIR}"
    infractl artifacts "${CLUSTER_NAME}" -d "${ARTIFACTS_DIR}" > /dev/null
}

function fetch_cluster_credentials() {
    read -ra DATA < <(kubectl get secret/access-rhacs -n stackrox -o jsonpath='{.data}' 2>/dev/null | jq -r '(.central_url|@base64d) +" "+ (.username|@base64d) +" "+(.password|@base64d)')
}

download_cluster_artifacts
fetch_cluster_credentials

echo -e "Saved cluster artifacts at '${ARTIFACTS_DIR}'.

To use the downloaded kubeconfig, run:

\t${C_PURPLE}export KUBECONFIG=${KUBECONFIG}${C_OFF}

Access Central at ${C_PURPLE}${DATA[0]}${C_OFF}.
Login with user '${C_PURPLE}${DATA[1]}${C_OFF}', password '${C_PURPLE}${DATA[2]}${C_OFF}'

To access the observability stack in this cluster, forward the port of Grafana.${C_OFF}
Example to copy & paste:${C_OFF}

\t${C_PURPLE}kubectl --kubeconfig ${KUBECONFIG} -n stackrox port-forward service/monitoring 48443:8443${C_OFF}

Then access at ${C_PURPLE}https://localhost:48443${C_OFF}, with with user '${C_PURPLE}admin${C_OFF}', password '${C_PURPLE}stackrox${C_OFF}'."
