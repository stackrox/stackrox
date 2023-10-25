#!/usr/bin/env bash

export C_YELLOW='\033[0;33m'
export C_PURPLE='\033[0;35m'
export C_OFF='\033[0m'

download_cluster_artifacts() {
    if [ "$(infractl whoami)" = "Anonymous" ]; then
        echo "ERROR: Ensure that your infractl is authenticated."
        exit 1
    fi
    if ! infractl list --all | grep -F "${CLUSTER_NAME}" > /dev/null; then
        echo "ERROR: Requested cluster does not exist."
        exit 1
    fi
    mkdir -p "${ARTIFACTS_DIR}"
    infractl artifacts "${CLUSTER_NAME}" -d "${ARTIFACTS_DIR}" > /dev/null 2>&1
}

fetch_cluster_credentials() {
    if ! command -v readarray > /dev/null 2>&1; then
        echo "ERROR: readarray is a requirement for this script. Ensure that your default Bash is at least v4."
        exit 1
    fi
    readarray -d ' ' -t DATA < <(kubectl get secret/access-rhacs -n stackrox -o jsonpath='{.data}' | jq -r '(.central_url|@base64d) +" "+ (.username|@base64d) +" "+(.password|@base64d)' | tr -d "\n")
}

print_cluster_credentials() {
    echo -e "Saved cluster artifacts at '${ARTIFACTS_DIR}'.

To use the downloaded kubeconfig, run:

\t${C_PURPLE}export KUBECONFIG=${KUBECONFIG}${C_OFF}

Access Central at ${C_PURPLE}${DATA[0]}${C_OFF}.
Login with user '${C_PURPLE}${DATA[1]}${C_OFF}', password '${C_PURPLE}${DATA[2]}${C_OFF}'."
}
