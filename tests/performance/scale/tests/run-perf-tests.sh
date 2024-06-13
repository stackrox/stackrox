#!/usr/bin/env bash
set -eoux pipefail

# Set number of pods per node
#oc create --filename=$HOME/stackrox/tests/performance/scale/utilities/examples/set-max-pods.yml

cd ${HOME}/stackrox/tests/performance/scale/utilities
./start-central-and-scanner.sh "${ARTIFACTS_DIR}"
./wait-for-pods.sh "${ARTIFACTS_DIR}"
./get-bundle.sh "${ARTIFACTS_DIR}"
./start-secured-cluster.sh $ARTIFACTS_DIR


cd ${HOME}/stackrox/tests/performance/scale/tests/kube-burner/cluster-density

./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 10 --num-deployments 5 --num-pods 1
./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 100 --num-deployments 5 --num-pods 1
./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 200 --num-deployments 5 --num-pods 1
./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 500 --num-deployments 5 --num-pods 1

./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 1250 --num-deployments 20 --num-pods 1
./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 1000 --num-deployments 6 --num-pods 4
./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 800 --num-deployments 10 --num-pods 3
./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 950 --num-deployments 9 --num-pods 3
