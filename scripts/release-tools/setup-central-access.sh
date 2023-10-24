#!/usr/bin/env bash

set -ueo pipefail

CWD=$(dirname "$0")
# shellcheck source=./lib.sh
source "${CWD}/lib.sh"

if [[ $# -eq 0 ]]; then
    echo -e "${C_YELLOW}Usage: $0 <CLUSTER_NAME>${C_OFF}"
    exit 1
fi

# Ensure that the cluster name is correct even if the user specified the version tag instead of the dashed name.
# shellcheck disable=2034
CLUSTER_NAME="${1//./-}"
ARTIFACTS_DIR="$(mktemp -d)/artifacts"
export KUBECONFIG="${ARTIFACTS_DIR}/kubeconfig"

download_cluster_artifacts
fetch_cluster_credentials
print_cluster_credentials

echo -e "To access the observability stack in this cluster, forward the port of Grafana.${C_OFF}
Example to copy & paste:${C_OFF}

\t${C_PURPLE}kubectl --kubeconfig ${KUBECONFIG} -n stackrox port-forward service/monitoring 48443:8443${C_OFF}

Then access at ${C_PURPLE}https://localhost:48443${C_OFF}, with with user '${C_PURPLE}admin${C_OFF}', password '${C_PURPLE}stackrox${C_OFF}'."
