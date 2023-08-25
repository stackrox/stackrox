#!/bin/bash

set -euo pipefail

PREVIOUS_RELEASE=$1
RELEASE=$2

CWD="$(pwd)"
TMP_DIR="$(mktemp -d)"

RELEASE_DIR="$TMP_DIR/${RELEASE}"
mkdir -p "${RELEASE_DIR}" && cd "${RELEASE_DIR}"

get_infra_artifacts() {
    infractl artifacts "upgrade-test1-${RELEASE//./-}" --download-dir=./artifacts/test1
    infractl artifacts "upgrade-test2-${RELEASE//./-}" --download-dir=./artifacts/test2
}

deploy_central() {
    OS="$(uname)"

    curl "https://mirror.openshift.com/pub/rhacs/assets/${PREVIOUS_RELEASE}/bin/${OS}/roxctl" --output "roxctl-${PREVIOUS_RELEASE}"
    chmod +x "roxctl-${PREVIOUS_RELEASE}"

    ./artifacts/test1/connect

    rm -rf bundle-test1
    ./roxctl-"${PREVIOUS_RELEASE}" central generate k8s pvc \
        --lb-type lb \
        --enable-pod-security-policies=false \
        --image-defaults development_build \
        --output-dir bundle-test1

    ./bundle-test1/central/scripts/setup.sh
    kubectl apply -R -f bundle-test1/central
    ./bundle-test1/scanner/scripts/setup.sh
    kubectl apply -R -f bundle-test1/scanner

    kubectl rollout status deployment central -n stackrox --watch --timeout 5m

    ROX_PASSWORD="$(cat bundle-test1/password)"
    echo "::add-mask::$ROX_PASSWORD"
    export ROX_PASSWORD

    COUNTER=0
    while [[ -z $(kubectl -n stackrox get service/central-loadbalancer -o jsonpath="{.status.loadBalancer.ingress}" 2>/dev/null) ]]; do
        if [ "$COUNTER" -lt "10" ]; then
            echo "Waiting for service/central-loadbalancer to get ingress ..."
            ((COUNTER++))
            sleep 30
        else
            echo "Timeout waiting for service/central-loadbalancer to get ingress!"
            exit 1
        fi
    done
    CENTRAL_IP="$(kubectl -n stackrox get service/central-loadbalancer -o json | jq -r '.status.loadBalancer.ingress[] | .ip')"
    echo "::add-mask::$CENTRAL_IP"
    export CENTRAL_IP
}

deploy_sensor() {
    CLUSTER_NAME=$1
    CENTRAL_API_ENDPOINT=$2
    COLLECTION_METHOD=$3

    # Register cluster
    CLUSTER_ID="$(curl "https://${CENTRAL_IP}/v1/clusters" \
        --insecure \
        --user "admin:${ROX_PASSWORD}" \
        --silent \
        --data-raw '{"name":"'"${CLUSTER_NAME}"'","type":"KUBERNETES_CLUSTER","mainImage":"quay.io/rhacs-eng/main","collectorImage":"quay.io/rhacs-eng/collector","centralApiEndpoint":"'"${CENTRAL_API_ENDPOINT}"'","runtimeSupport":false,"collectionMethod":"'"${COLLECTION_METHOD}"'","DEPRECATEDProviderMetadata":null,"admissionControllerEvents":true,"admissionController":false,"admissionControllerUpdates":false,"DEPRECATEDOrchestratorMetadata":null,"tolerationsConfig":{"disabled":false},"dynamicConfig":{"admissionControllerConfig":{"enabled":false,"enforceOnUpdates":false,"timeoutSeconds":3,"scanInline":false,"disableBypass":false},"registryOverride":""},"slimCollector":true}' \
        | jq -r '.cluster.id'
    )"

    # Download sensor bundle
    curl "https://${CENTRAL_IP}/api/extensions/clusters/zip" \
        --insecure \
        --user "admin:${ROX_PASSWORD}" \
        --data-raw '{"id":"'"${CLUSTER_ID}"'","createUpgraderSA":true}' \
        --output "sensor-${CLUSTER_NAME}.zip"

    "./artifacts/${CLUSTER_NAME}/connect"
    unzip -d "sensor-${CLUSTER_NAME}" "sensor-${CLUSTER_NAME}.zip"

    rm ./sensor-"${CLUSTER_NAME}"/*-pod-security.yaml

    "./sensor-${CLUSTER_NAME}/sensor.sh"
}

disable_autoupgrader() {
    curl "https://${CENTRAL_IP}/v1/sensorupgrades/config" \
        --insecure \
        --user "admin:${ROX_PASSWORD}" \
        --data-raw '{"config":{"enableAutoUpgrades":false}}'
}

deploy_violations() {
    CLUSTER_NAME=$1
    "./artifacts/${CLUSTER_NAME}/connect"
    kubectl apply -f "${CWD}/.github/static/upgrade-test/violations.yaml"
}

create_policy() {
    curl "https://${CENTRAL_IP}/v1/policies?enableStrictValidation=true" \
        --user "admin:${ROX_PASSWORD}" \
        --insecure \
        --data @"${CWD}"/.github/static/upgrade-test/policy.json | jq -r '.id'
}

trigger_compliance_check() {
    curl "https://${CENTRAL_IP}/api/graphql?opname=triggerScan" \
        --user "admin:${ROX_PASSWORD}" \
        --insecure \
        --data-raw $'{"operationName":"triggerScan","variables":{"clusterId":"*","standardId":"*"},"query":"mutation triggerScan($clusterId: ID\u0021, $standardId: ID\u0021) {\\n  complianceTriggerRuns(clusterId: $clusterId, standardId: $standardId) {\\n    id\\n    standardId\\n    clusterId\\n    state\\n    errorMessage\\n    __typename\\n  }\\n}\\n"}'
}

save_credentials_to_cluster() {
    cat <<EOF | kubectl -n stackrox apply -f -
    apiVersion: v1
    kind: Secret
    metadata:
      name: access-rhacs
      namespace: stackrox
    data:
      central_url: "$(echo https://"${CENTRAL_IP}" | base64)"
      password: "$(echo "${ROX_PASSWORD}" | base64)"
      username: "$(echo "admin" | base64)"
EOF
}

get_infra_artifacts
deploy_central
save_credentials_to_cluster
disable_autoupgrader
deploy_violations "test1"
deploy_violations "test2"
deploy_sensor "test1" "central.stackrox:443" "EBPF"
deploy_sensor "test2" "${CENTRAL_IP}:443" "EBPF"
create_policy
trigger_compliance_check
