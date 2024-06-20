#!/usr/bin/env bash
set -eoux pipefail

# Set number of pods per node


cd ${HOME}/stackrox/tests/performance/scale/utilities
./start-central-and-scanner.sh "${ARTIFACTS_DIR}"
./wait-for-pods.sh "${ARTIFACTS_DIR}"
./get-bundle.sh "${ARTIFACTS_DIR}"
./start-secured-cluster.sh $ARTIFACTS_DIR

#oc -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"32Gi","cpu":"16"},"limits":{"memory":"32Gi","cpu":"16"}}}]}}}}'
oc -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"8Gi","cpu":"4"},"limits":{"memory":"8Gi","cpu":"4"}}}]}}}}'
oc -n stackrox patch deploy/scanner-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"db","resources":{"requests":{"memory":"8Gi","cpu":"4"},"limits":{"memory":"8Gi","cpu":"4"}}}]}}}}'
oc -n stackrox patch deploy/scanner -p '{"spec":{"template":{"spec":{"containers":[{"name":"scanner","resources":{"requests":{"memory":"8Gi","cpu":"4"},"limits":{"memory":"8Gi","cpu":"4"}}}]}}}}'
oc -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"16Gi","cpu":"8"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
oc patch deployment central-db --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/volumes/4/emptyDir/sizeLimit", "value": "4Gi"}]' --namespace=stackrox

cd ${HOME}/stackrox/tests/performance/scale/tests/kube-burner/cluster-density

#./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 10 --num-deployments 5 --num-pods 1
#./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 100 --num-deployments 5 --num-pods 1
#./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 200 --num-deployments 5 --num-pods 1
#./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 500 --num-deployments 5 --num-pods 1

./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 1250 --num-deployments 20 --num-pods 1
./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 1000 --num-deployments 6 --num-pods 4
./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 800 --num-deployments 10 --num-pods 3
./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 950 --num-deployments 9 --num-pods 3
