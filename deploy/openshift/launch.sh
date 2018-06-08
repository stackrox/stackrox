#!/usr/bin/env bash

function launch_central {
    ROX_CENTRAL_DASHBOARD_PORT="$1"
    LOCAL_API_ENDPOINT="$2"
    OPENSHIFT_DIR="$3"
    PREVENT_IMAGE="$4"
    NAMESPACE="$5"

    set -u

    echo "Generating central config..."
    docker run "$PREVENT_IMAGE" deploy openshift -n "$NAMESPACE" -i "$PREVENT_IMAGE" > $OPENSHIFT_DIR/central.zip
    UNZIP_DIR="OPENSHIFT_DIR/central-deploy/"
    rm -rf "$UNZIP_DIR"
    unzip "OPENSHIFT_DIR/central.zip" -d "$UNZIP_DIR"
    echo

    echo "Deploying Central..."
    $UNZIP_DIR/deploy.sh
    echo

    $UNZIP_DIR/port-forward.sh 8000
    export LOCAL_API_ENDPOINT=localhost:8000
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

    COMMON_PARAMS="{ \"params\" : { \"namespace\": \"$NAMESPACE\", \"benchmarkServiceAccount\":\"$BENCHMARK_SERVICE_ACCOUNT\" } }"

    EXTRA_CONFIG="\"openshift\": $COMMON_PARAMS }"

    get_cluster_zip "$LOCAL_API_ENDPOINT" "$CLUSTER" OPENSHIFT_CLUSTER "$PREVENT_IMAGE" "$CLUSTER_API_ENDPOINT" "$K8S_DIR" "$EXTRA_CONFIG"

    echo "Deploying Sensor..."
    UNZIP_DIR="$K8S_DIR/sensor-deploy/"
    rm -rf "$UNZIP_DIR"
    unzip "$K8S_DIR/sensor-deploy.zip" -d "$UNZIP_DIR"
    $UNZIP_DIR/deploy.sh
    echo
}
