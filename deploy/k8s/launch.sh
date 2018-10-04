#!/usr/bin/env bash

function launch_central {
    K8S_DIR="$1"
    PREVENT_IMAGE="$2"

    flags=()
    if [[ "${ROX_RUNTIME_POLICIES}" = "true" ]]; then
      flags=(--flags ROX_RUNTIME_POLICIES)
    fi

    set -u

    echo "Generating central config..."
    docker run --rm "$PREVENT_IMAGE" ${flags[@]+"${flags[@]}"} deploy k8s -n stackrox -i "$PREVENT_IMAGE" none > $K8S_DIR/central.zip
    UNZIP_DIR="$K8S_DIR/central-deploy/"
    rm -rf "$UNZIP_DIR"
    unzip "$K8S_DIR/central.zip" -d "$UNZIP_DIR"
    echo

    echo "Deploying Central..."
    $UNZIP_DIR/central.sh
    echo

    $UNZIP_DIR/port-forward.sh 8000
    wait_for_central "localhost:8000"
    echo "Successfully deployed Central!"
    echo "Access the UI at: https://localhost:8000"
}

function launch_sensor {
    K8S_DIR="$1"
    CLUSTER="$2"
    PREVENT_IMAGE="$3"
    CLUSTER_API_ENDPOINT="$4"
    RUNTIME_SUPPORT="$5"

    COMMON_PARAMS="{ \"params\" : { \"namespace\": \"stackrox\" }, \"imagePullSecret\": \"stackrox\" }"

    EXTRA_CONFIG="\"kubernetes\": $COMMON_PARAMS }"

    get_cluster_zip localhost:8000 "$CLUSTER" KUBERNETES_CLUSTER "$PREVENT_IMAGE" "$CLUSTER_API_ENDPOINT" "$K8S_DIR" "$RUNTIME_SUPPORT" "$EXTRA_CONFIG"

    echo "Deploying Sensor..."
    UNZIP_DIR="$K8S_DIR/sensor-deploy/"
    rm -rf "$UNZIP_DIR"
    unzip "$K8S_DIR/sensor-deploy.zip" -d "$UNZIP_DIR"
    $UNZIP_DIR/sensor.sh
    echo
}
