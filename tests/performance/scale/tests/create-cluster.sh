#!/usr/bin/env bash
set -eou pipefail

export INFRA_NAME=$1

export ARTIFACTS_DIR="/tmp/artifacts-${INFRA_NAME}"

infractl create openshift-4-perf-scale "${INFRA_NAME}" --arg master-node-type=n2-standard-16 --description "Perf testing cluster" --download-dir="${ARTIFACTS_DIR}"

export KUBECONFIG="${ARTIFACTS_DIR}/kubeconfig"

infractl lifespan "${INFRA_NAME}" 24h

# Set number of pods per node
oc create --filename=../utilities/examples/set-max-pods.yml

for machineset in `oc get machineset.machine.openshift.io --namespace openshift-machine-api  | tail -n +2 | awk '{print $1}'`; do
	oc scale --replicas=18 machineset --namespace openshift-machine-api $machineset
done

./isolate-monitoring.sh
