#!/usr/bin/env bash
set -eoux pipefail

export INFRA_NAME=$1

export ARTIFACTS_DIR="/tmp/artifacts-${INFRA_NAME}"

infractl create openshift-4-perf-scale "${INFRA_NAME}" --arg master-node-type=n2-standard-16 --arg worker-node-type=c2-standard-8 --description "Perf testing cluster" --download-dir="${ARTIFACTS_DIR}"
#infractl create openshift-4-perf-scale "${INFRA_NAME}" --arg master-node-type=n2-standard-16 --arg worker-node-type=c2-standard-8 --description "Perf testing cluster" --download-dir="${ARTIFACTS_DIR}" --arg openshift-version=ocp/4.11.0
#infractl create openshift-4-perf-scale "${INFRA_NAME}" --arg master-node-type=n2-standard-16 --description "Perf testing cluster" --download-dir="${ARTIFACTS_DIR}" --arg openshift-version=ocp/4.11.0

export KUBECONFIG="${ARTIFACTS_DIR}/kubeconfig"

infractl lifespan "${INFRA_NAME}" 24h

stackrox_dir="$(git rev-parse --show-toplevel || echo "$HOME/go/src/github.com/stackrox/stackrox")"

# Set number of pods per node
max_pods_set="$(oc get KubeletConfig set-max-pods || true)"
if [[ -z "$max_pods_set" ]]; then
  oc create --filename="$stackrox_dir/tests/performance/scale/utilities/examples/set-max-pods.yml"
fi

for machineset in `oc get machineset.machine.openshift.io --namespace openshift-machine-api  | tail -n +2 | awk '{print $1}'`; do
	oc scale --replicas=20 machineset --namespace openshift-machine-api $machineset
done

./isolate-monitoring.sh


echo "export KUBECONFIG=$KUBECONFIG"
