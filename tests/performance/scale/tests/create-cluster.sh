#!/usr/bin/env bash
set -eoux pipefail

export INFRA_NAME=$1
utilities_dir=$2
num_replicas=${3:-3}
lifespan=${4:-24h}

DIR="$(cd "$(dirname "$0")" && pwd)"

does_cluster_exist() {
    error_code=0
    infractl get "$INFRA_NAME" &> /dev/null || error_code=$?
    echo "$error_code"
    #return "$(infractl get "$INFRA_NAME" &> /dev/null; echo $?)"
}

export ARTIFACTS_DIR="/tmp/artifacts-${INFRA_NAME}"

if [ "$(does_cluster_exist)" -eq 0 ]; then
    echo "A cluster with the name '${INFRA_NAME}' already exists"
else
    infractl create openshift-4-perf-scale "${INFRA_NAME}" --arg master-node-type=n2-standard-16 --arg worker-node-type=c2-standard-8 --description "Perf testing cluster" --download-dir="${ARTIFACTS_DIR}"
    infractl lifespan "${INFRA_NAME}" "$lifespan"
fi

export KUBECONFIG="${ARTIFACTS_DIR}/kubeconfig"
echo "KUBECONFIG=$KUBECONFIG"


# Set number of pods per node
max_pods_set="$(oc get KubeletConfig set-max-pods || true)"
if [[ -z "$max_pods_set" ]]; then
  oc create --filename="${utilities_dir}"/examples/set-max-pods.yml
fi

mapfile -t machinesets < <(oc get machineset.machine.openshift.io --namespace openshift-machine-api  | tail -n +2 | awk '{print $1}')
for machineset in "${machinesets[@]}"; do
	oc scale --replicas="${num_replicas}" machineset --namespace openshift-machine-api "$machineset"
done

"${DIR}/isolate-monitoring.sh"
