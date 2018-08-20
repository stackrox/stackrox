#!/usr/bin/env bash

function launch_central {
    LOCAL_API_ENDPOINT="$1"
    OPENSHIFT_DIR="$2"
    PREVENT_IMAGE_REPO="$3"
    PREVENT_IMAGE_TAG="$4"

    set -u

    echo "Generating central config..."
    docker run "$PREVENT_IMAGE" deploy openshift -n stackrox -i "$PREVENT_IMAGE_REPO/stackrox/prevent:$PREVENT_IMAGE_TAG" > $OPENSHIFT_DIR/central.zip
    UNZIP_DIR="$OPENSHIFT_DIR/central-deploy/"
    rm -rf "$UNZIP_DIR"
    unzip "$OPENSHIFT_DIR/central.zip" -d "$UNZIP_DIR"
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
    K8S_DIR="$6"
    LOCAL_API_ENDPOINT="$1"
    CLUSTER="$2"
    PREVENT_IMAGE_TAG="$3"
    CLUSTER_API_ENDPOINT="$4"
    RUNTIME_SUPPORT="$5"

    COMMON_PARAMS="{ \"params\" : { \"namespace\": \"stackrox\" } }"

    EXTRA_CONFIG="\"openshift\": $COMMON_PARAMS }"

    get_cluster_zip "$LOCAL_API_ENDPOINT" "$CLUSTER" OPENSHIFT_CLUSTER "docker-registry.default.svc:5000/stackrox/prevent:$PREVENT_IMAGE_TAG" "$CLUSTER_API_ENDPOINT" "$K8S_DIR" "$RUNTIME_SUPPORT" "$EXTRA_CONFIG"

    echo "Deploying Sensor..."
    UNZIP_DIR="$K8S_DIR/sensor-deploy/"
    rm -rf "$UNZIP_DIR"
    unzip "$K8S_DIR/sensor-deploy.zip" -d "$UNZIP_DIR"
    $UNZIP_DIR/deploy.sh
    echo
}
