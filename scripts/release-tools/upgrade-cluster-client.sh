#!/usr/bin/env bash

set -euo pipefail

# shellcheck source=./lib.sh
source "$(dirname "$0")/lib.sh"

if [[ $# -eq 0 ]]; then
    echo -e "${C_YELLOW}Usage: $0 <MILESTONE>${C_OFF}"
    exit 1
fi

RELEASE=$1
CLUSTER_NAME="upgrade-test1-${RELEASE//./-}"

ARTIFACTS_DIR="$(mktemp -d)/artifacts"
export KUBECONFIG="${ARTIFACTS_DIR}/kubeconfig"

download_cluster_artifacts
fetch_cluster_credentials
print_cluster_credentials

CLUSTER_NAME_SENSOR_ONLY="${CLUSTER_NAME//test1/test2}"
echo -e "To reproduce cluster access, run:

\t${C_PURPLE}infractl artifacts upgrade-test1-${RELEASE//./-} --download-dir ./artifacts/test1${C_OFF}

Use ${C_PURPLE}${CLUSTER_NAME_SENSOR_ONLY}${C_OFF} to access the sensor-only cluster.

-> Follow the instructions at ${C_PURPLE}https://docs.engineering.redhat.com/display/StackRox/Upgrade+test#Upgradetest-Upgradecomponents${C_OFF} to test the upgrade process."
