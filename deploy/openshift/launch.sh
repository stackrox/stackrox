#!/usr/bin/env bash

function launch_central {
    local openshift_dir="$1"
    local main_image="$2"

    set -u

    local extra_args=()
    if [[ "$MONITORING_SUPPORT" == "false" ]]; then
        extra_args+=("--monitoring-type=none")
    fi

    docker run  -e ROX_HTPASSWD_AUTH=${ROX_HTPASSWD_AUTH} "$main_image" deploy openshift ${extra_args[@]+"${extra_args[@]}"} --monitoring-password stackrox -i "$main_image" none > $openshift_dir/central.zip
    local unzip_dir="$openshift_dir/central-deploy/"
    rm -rf "${unzip_dir}"
    unzip "$openshift_dir/central.zip" -d "${unzip_dir}"
    echo

    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
        echo "Deploying Monitoring..."
        $unzip_dir/monitoring/scripts/setup.sh
        oc create -R -f $unzip_dir/monitoring
        echo

        oc -n stackrox patch deployment monitoring --patch "$(cat $K8S_DIR/monitoring-resources-patch.yaml)"
    fi

    echo "Deploying Central..."
    $unzip_dir/central/scripts/setup.sh
    oc create -R -f $unzip_dir/central
    echo

    $unzip_dir/central/scripts/port-forward.sh 8000
    local local_api_endpoint=localhost:8000
    echo "Set local API endpoint to: $local_api_endpoint"

    wait_for_central "$local_api_endpoint"
    echo "Successfully deployed Central!"
    echo "Access the UI at: https://$local_api_endpoint"
}

function launch_sensor {
    local openshift_dir="$1"
    local cluster="$2"
    local main_image="$3"
    local cluster_api_endpoint="$4"
    local runtime_support="$5"

    local common_params="{ \"params\" : { \"namespace\": \"stackrox\" } }"

    local extra_config=""
    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
        extra_config+='"monitoringEndpoint": "monitoring.stackrox:443", '
    fi
    extra_config+="\"openshift\": $common_params}"

    get_cluster_zip localhost:8000 "$cluster" OPENSHIFT_CLUSTER "$main_image" "$cluster_api_endpoint" "$openshift_dir" "$runtime_support" "$extra_config"

    echo "Deploying Sensor..."
    local unzip_dir="$openshift_dir/sensor-deploy/"
    rm -rf "$unzip_dir"
    unzip "$openshift_dir/sensor-deploy.zip" -d "$unzip_dir"
    $unzip_dir/sensor.sh
    echo
}
