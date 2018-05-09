#!/usr/bin/env bash

function create_registry_secrets {

    echo "Creating image pull secrets..."
    PULL_SECRET_NAME="stackrox"
    kubectl delete secret "$PULL_SECRET_NAME" --namespace "$NAMESPACE" || true

    if ! kubectl get secret -n $NAMESPACE | grep $PULL_SECRET_NAME | grep dockerconfigjson > /dev/null; then
      kubectl create secret docker-registry \
        "$PULL_SECRET_NAME" --namespace "$NAMESPACE" \
        --docker-server=https://index.docker.io/v1/ \
        --docker-username="$REGISTRY_USERNAME" \
        --docker-password="$REGISTRY_PASSWORD" \
        --docker-email="does-not-matter@stackrox.io"

      echo
    fi
}

function launch_central {
    K8S_DIR="$1"
    PREVENT_IMAGE="$2"
    NAMESPACE="$3"

    set -u

    create_registry_secrets

    echo "Generating central config..."
    docker run "$PREVENT_IMAGE" -t k8s -n "$NAMESPACE" -i "$PREVENT_IMAGE" > $K8S_DIR/central.zip
    UNZIP_DIR="$K8S_DIR/central-deploy/"
    rm -rf "$UNZIP_DIR"
    unzip "$K8S_DIR/central.zip" -d "$UNZIP_DIR"
    echo

    echo "Deploying Central..."
    kubectl delete secret -n "$NAMESPACE" central-tls || true
    kubectl delete -f "$UNZIP_DIR/deploy.yaml" || true
    $UNZIP_DIR/deploy.sh
    echo

    echo -n "Waiting for Central pod to be ready."
    until [ "$(kubectl get pod -n $NAMESPACE --selector 'app=central' | grep Running | wc -l)" -eq 1 ]; do
        echo -n .
        sleep 1
    done
    echo

    pkill -f "kubectl port-forward -n ${NAMESPACE}" || true
    export CENTRAL_POD="$(kubectl get pod -n $NAMESPACE --selector 'app=central' --output=jsonpath='{.items..metadata.name} {.items..status.phase}' | grep Running | cut -f 1 -d ' ')"
    kubectl port-forward -n "$NAMESPACE" "$CENTRAL_POD" 8000:443 &> /dev/null &
    PID="$!"
    echo "Port-forward launched with PID: $PID"
    LOCAL_API_ENDPOINT=localhost:8000
    echo "Set local API endpoint to: $LOCAL_API_ENDPOINT"

    wait_for_central "$LOCAL_API_ENDPOINT"
    echo "Successfully deployed Central!"
    echo "Access the UI at: https://$LOCAL_API_ENDPOINT"
}

function launch_sensor {
    LOCAL_API_ENDPOINT="$1"
    CLUSTER="$2"
    PREVENT_IMAGE="$3"
    CLUSTER_API_ENDPOINT="$4"
    K8S_DIR="$5"
    NAMESPACE="$6"
    BENCHMARK_SERVICE_ACCOUNT="${7-benchmark}"

    EXTRA_CONFIG="\"namespace\": \"$NAMESPACE\", \"imagePullSecret\": \"stackrox\", \"kubernetes\": { \"benchmarkServiceAccount\":\"$BENCHMARK_SERVICE_ACCOUNT\" } }"

    create_registry_secrets

    get_cluster_zip "$LOCAL_API_ENDPOINT" "$CLUSTER" KUBERNETES_CLUSTER "$PREVENT_IMAGE" "$CLUSTER_API_ENDPOINT" "$K8S_DIR" "$EXTRA_CONFIG"

    echo "Deploying Sensor..."
    kubectl delete secret -n "$NAMESPACE" sensor-tls || true
    UNZIP_DIR="$K8S_DIR/sensor-deploy/"
    rm -rf "$UNZIP_DIR"
    unzip "$K8S_DIR/sensor-deploy.zip" -d "$UNZIP_DIR"
    $UNZIP_DIR/sensor-deploy.sh
    echo
}
