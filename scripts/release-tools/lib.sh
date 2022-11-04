#!/usr/bin/env bash

export C_YELLOW='\033[0;33m'
export C_PURPLE='\033[0;35m'
export C_OFF='\033[0m'

download_cluster_artifacts() {
    mkdir -p "${ARTIFACTS_DIR}"
    infractl artifacts "${CLUSTER_NAME}" -d "${ARTIFACTS_DIR}" > /dev/null 2>&1
}

fetch_cluster_credentials() {
    readarray -t DATA < <(kubectl get secret/access-rhacs -n stackrox -o jsonpath='{.data}' 2>/dev/null | jq -r '(.central_url|@base64d) +" "+ (.username|@base64d) +" "+(.password|@base64d)')
}

print_cluster_credentials() {
    echo -e "Saved cluster artifacts at '${ARTIFACTS_DIR}'.

To use the downloaded kubeconfig, run:

\t${C_PURPLE}export KUBECONFIG=${KUBECONFIG}${C_OFF}

Access Central at ${C_PURPLE}${DATA[0]}${C_OFF}.
Login with user ${C_PURPLE}${DATA[1]}${C_OFF}, password ${C_PURPLE}${DATA[2]}${C_OFF}."
}
