#!/usr/bin/env bash

function launch_central {
    OPENSHIFT_DIR="$1"
    PREVENT_IMAGE="$2"

    set -u

    EXTRA_ARGS=()
    if [[ "$MONITORING_SUPPORT" == "false" ]]; then
        EXTRA_ARGS+=("--monitoring-type=none")
    fi

    docker run "$PREVENT_IMAGE" deploy openshift ${EXTRA_ARGS[@]+"${EXTRA_ARGS[@]}"} -i "$PREVENT_IMAGE" none > $OPENSHIFT_DIR/central.zip
    UNZIP_DIR="$OPENSHIFT_DIR/central-deploy/"
    rm -rf "$UNZIP_DIR"
    unzip "$OPENSHIFT_DIR/central.zip" -d "$UNZIP_DIR"
    echo

    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
        echo "Deploying Monitoring..."
        $UNZIP_DIR/monitoring/monitoring.sh
        echo

        oc -n stackrox patch deployment monitoring --patch "$(cat $K8S_DIR/monitoring-resources-patch.yaml)"
    fi

    echo "Deploying Central..."
    $UNZIP_DIR/central.sh
    echo

    $UNZIP_DIR/port-forward.sh 8000
    export LOCAL_API_ENDPOINT=localhost:8000
    echo "Set local API endpoint to: $LOCAL_API_ENDPOINT"

    wait_for_central "$LOCAL_API_ENDPOINT"
    echo "Successfully deployed Central!"
    echo "Access the UI at: https://$LOCAL_API_ENDPOINT"
}

function launch_sensor {
    OPENSHIFT_DIR="$1"
    CLUSTER="$2"
    PREVENT_IMAGE="$3"
    CLUSTER_API_ENDPOINT="$4"
    RUNTIME_SUPPORT="$5"

    COMMON_PARAMS="{ \"params\" : { \"namespace\": \"stackrox\" } }"

    EXTRA_CONFIG=""
    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
        EXTRA_CONFIG+='"monitoringEndpoint": "monitoring.stackrox", '
    fi
    EXTRA_CONFIG+="\"openshift\": $COMMON_PARAMS }"

    get_cluster_zip localhost:8000 "$CLUSTER" OPENSHIFT_CLUSTER "$PREVENT_IMAGE" "$CLUSTER_API_ENDPOINT" "$OPENSHIFT_DIR" "$RUNTIME_SUPPORT" "$EXTRA_CONFIG"

    echo "Deploying Sensor..."
    UNZIP_DIR="$OPENSHIFT_DIR/sensor-deploy/"
    rm -rf "$UNZIP_DIR"
    unzip "$OPENSHIFT_DIR/sensor-deploy.zip" -d "$UNZIP_DIR"
    $UNZIP_DIR/sensor.sh
    echo
}
