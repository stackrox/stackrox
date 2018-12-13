#!/usr/bin/env bash

function roxctl_cmd {
    if [[ -x "$(command -v roxctl)" && "$(roxctl version)" == "$MAIN_IMAGE_TAG" ]]; then
       roxctl $@
    else
       docker run --rm -e ROX_HTPASSWD_AUTH "$MAIN_IMAGE" $@
    fi
}

function launch_central {
    local openshift_dir="$1"

    set -u

    local extra_args=()
    if [[ "$MONITORING_SUPPORT" == "false" ]]; then
        extra_args+=("--monitoring-type=none")
    fi

    roxctl_cmd central generate openshift ${extra_args[@]+"${extra_args[@]}"} --monitoring-password stackrox -i "$MAIN_IMAGE" none > $openshift_dir/central.zip
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

	if [[ -f "${unzip_dir}/password" ]]; then
		export ROX_ADMIN_USER=admin
		export ROX_ADMIN_PASSWORD="$(< "${unzip_dir}/password")"
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

    local common_params="{ \"params\" : { \"namespace\": \"stackrox\" } }"

    local extra_config=""
    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
        extra_config+='"monitoringEndpoint": "monitoring.stackrox:443", '
    fi
    extra_config+="\"openshift\": $common_params}"

    get_cluster_zip localhost:8000 "$CLUSTER" OPENSHIFT_CLUSTER "$MAIN_IMAGE" "$CLUSTER_API_ENDPOINT" "$openshift_dir" "$RUNTIME_SUPPORT" "$extra_config"

    echo "Deploying Sensor..."
    local unzip_dir="$openshift_dir/sensor-deploy/"
    rm -rf "$unzip_dir"
    unzip "$openshift_dir/sensor-deploy.zip" -d "$unzip_dir"
    $unzip_dir/sensor.sh
    echo
}
