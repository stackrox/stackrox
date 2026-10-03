#!/usr/bin/env bash
set -eoux pipefail


json_tests_file=~/perf-tests.json

utilities_dir=${HOME}/stackrox/tests/performance/scale/utilities



json_tests="$(jq .perfTests "$json_tests_file" --raw-output)"

echo "$json_tests" | jq

cd ${HOME}/stackrox/tests/performance/scale/tests/kube-burner/cluster-density

ntests="$(jq '.perfTests | length' "$json_tests_file")"
for ((i = 0; i < ntests; i = i + 1)); do
        printf 'yes\n'  | "${HOME}/workflow/bin/teardown"

        "${utilities_dir}/start-central-and-scanner.sh" "${ARTIFACTS_DIR}"
        "${utilities_dir}/wait-for-pods.sh" "${ARTIFACTS_DIR}"
        "${utilities_dir}/get-bundle.sh" "${ARTIFACTS_DIR}"
        "${utilities_dir}/start-secured-cluster.sh" $ARTIFACTS_DIR "$COLLECTOR_IMAGE_TAG" "$COLLECTOR_IMAGE_REGISTRY"
	#${HOME}/stackrox/deploy/deploy.sh
	#sleep 120
        oc -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"8Gi","cpu":"4"},"limits":{"memory":"8Gi","cpu":"4"}}}]}}}}'
        oc -n stackrox patch deploy/scanner-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"db","resources":{"requests":{"memory":"8Gi","cpu":"4"},"limits":{"memory":"8Gi","cpu":"4"}}}]}}}}'
        oc -n stackrox patch deploy/scanner -p '{"spec":{"template":{"spec":{"containers":[{"name":"scanner","resources":{"requests":{"memory":"8Gi","cpu":"4"},"limits":{"memory":"8Gi","cpu":"4"}}}]}}}}'
        oc -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"16Gi","cpu":"8"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
        oc -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"requests":{"memory":"16Gi","cpu":"8"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
        oc patch deployment central-db --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/volumes/4/emptyDir/sizeLimit", "value": "4Gi"}]' --namespace=stackrox
        sleep 1800

        version="$(jq .perfTests "$json_tests_file" | jq .["$i"])"
        num_namespaces="$(echo $version | jq .numNamespaces)"
        num_deployments="$(echo $version | jq .numDeployments)"
        num_pods="$(echo $version | jq .numPods)"
        echo "num_namespaces= $num_namespaces"
        echo "num_deployments= $num_deployments"
        echo "num_pods= $num_pods"
        ./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces "${num_namespaces}" --num-deployments "${num_deployments}" --num-pods "${num_pods}"
done
