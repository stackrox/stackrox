#!/usr/bin/env bash

function launch_central {
    local k8s_dir="$1"
    local main_image="$2"
    local storage="$3"

    echo "Generating central config..."

    EXTRA_ARGS=()
    if [[ "$MONITORING_SUPPORT" == "false" ]]; then
        EXTRA_ARGS+=("--monitoring-type=none")
    fi

    docker run --rm "${main_image}" deploy k8s ${EXTRA_ARGS[@]} -i "$main_image" "${storage}" > "${k8s_dir}/central.zip"

    local unzip_dir="${k8s_dir}/central-deploy/"
    rm -rf "${unzip_dir}"
    unzip "${k8s_dir}/central.zip" -d "${unzip_dir}"
    echo

    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
        echo "Deploying Monitoring..."
        $unzip_dir/monitoring/monitoring.sh
        echo

        kubectl -n stackrox patch deployment monitoring --patch "$(cat $k8s_dir/monitoring-resources-patch.yaml)"
    fi

    echo "Deploying Central..."
    $unzip_dir/central.sh
    echo

    $unzip_dir/port-forward.sh 8000
    wait_for_central "localhost:8000"
    echo "Successfully deployed Central!"
    echo "Access the UI at: https://localhost:8000"
}

function launch_sensor {
    local k8s_dir="$1"
    local cluster="$2"
    local main_image="$3"
    local cluster_api_endpoint="$4"
    local runtime_support="$5"

    local common_params="{ \"params\" : { \"namespace\": \"stackrox\" }, \"imagePullSecret\": \"stackrox\" }"

    local extra_config=""
    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
        extra_config+='"monitoringEndpoint": "monitoring.stackrox:8086", '
    fi
    extra_config+="\"kubernetes\": $common_params }"

    get_cluster_zip localhost:8000 "$cluster" KUBERNETES_CLUSTER "$main_image" "$cluster_api_endpoint" "$k8s_dir" "$runtime_support" "$extra_config"

    echo "Deploying Sensor..."
    local unzip_dir="$k8s_dir/sensor-deploy/"
    rm -rf "$unzip_dir"
    unzip "$k8s_dir/sensor-deploy.zip" -d "$unzip_dir"
    $unzip_dir/sensor.sh
    echo

}
