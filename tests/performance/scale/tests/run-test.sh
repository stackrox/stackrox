#!/usr/bin/env bash
set -eou pipefail

num_namespaces=$1
num_deployments=$2
num_pods=$3
cluster_name=$4

DIR="$(cd "$(dirname "$0")" && pwd)"

utilities_dir="${DIR}/../utilities"

source "${DIR}"/create-cluster "$cluster_name"

printf 'yes\n'  | "${HOME}/workflow/bin/teardown"

"${utilities_dir}/start-central-and-scanner.sh" "${ARTIFACTS_DIR}"
"${utilities_dir}/wait-for-pods.sh" "${ARTIFACTS_DIR}"
"${utilities_dir}/get-bundle.sh" "${ARTIFACTS_DIR}"
"${utilities_dir}/start-secured-cluster.sh" $ARTIFACTS_DIR "$COLLECTOR_IMAGE_TAG" "$COLLECTOR_IMAGE_REGISTRY"

oc -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"8Gi","cpu":"4"},"limits":{"memory":"8Gi","cpu":"4"}}}]}}}}'
oc -n stackrox patch deploy/scanner-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"db","resources":{"requests":{"memory":"8Gi","cpu":"4"},"limits":{"memory":"8Gi","cpu":"4"}}}]}}}}'
oc -n stackrox patch deploy/scanner -p '{"spec":{"template":{"spec":{"containers":[{"name":"scanner","resources":{"requests":{"memory":"8Gi","cpu":"4"},"limits":{"memory":"8Gi","cpu":"4"}}}]}}}}'
oc -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"16Gi","cpu":"8"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
oc -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"requests":{"memory":"16Gi","cpu":"8"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
oc patch deployment central-db --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/volumes/4/emptyDir/sizeLimit", "value": "4Gi"}]' --namespace=stackrox

"${DIR}/kube-burner/cluster-density/run-workload.sh" --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces "${num_namespaces}" --num-deployments "${num_deployments}" --num-pods "${num_pods}"

infractl delete "$cluster_name"
