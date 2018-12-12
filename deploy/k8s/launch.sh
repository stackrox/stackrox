#!/usr/bin/env bash

function launch_service {
    local dir="$1"
    local service="$2"

    if [[ "${OUTPUT_FORMAT}" == "helm" ]]; then
        helm install "$dir/$service" --name $service
    else
        kubectl create -R -f "$dir/$service"
    fi
}

function launch_central {
    local k8s_dir="$1"

    echo "Generating central config..."

    EXTRA_ARGS=()
    if [[ "$MONITORING_SUPPORT" == "false" ]]; then
        EXTRA_ARGS+=("--monitoring-type=none")
    else
        EXTRA_ARGS+=("--monitoring-lb-type=$MONITORING_LOAD_BALANCER")
    fi
    EXTRA_ARGS+=("--lb-type=$LOAD_BALANCER")

    docker run --rm -e ROX_HTPASSWD_AUTH "${MAIN_IMAGE}" central generate k8s ${EXTRA_ARGS[@]} --output-format="${OUTPUT_FORMAT}" \
     --monitoring-password stackrox -i "${MAIN_IMAGE}" "${STORAGE}" > "${k8s_dir}/central.zip"

    local unzip_dir="${k8s_dir}/central-deploy/"
    rm -rf "${unzip_dir}"
    unzip "${k8s_dir}/central.zip" -d "${unzip_dir}"

    echo

    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
        echo "Deploying Monitoring..."
        $unzip_dir/monitoring/scripts/setup.sh
        launch_service $unzip_dir monitoring
        echo

        kubectl -n stackrox patch deployment monitoring --patch "$(cat $k8s_dir/monitoring-resources-patch.yaml)"
    fi

	if [[ -f "${unzip_dir}/password" ]]; then
		export ROX_ADMIN_USER=admin
		export ROX_ADMIN_PASSWORD="$(< "${unzip_dir}/password")"
	fi

    echo "Deploying Central..."
    $unzip_dir/central/scripts/setup.sh
    launch_service $unzip_dir central
    echo

    $unzip_dir/central/scripts/port-forward.sh 8000
    wait_for_central "localhost:8000"
    echo "Successfully deployed Central!"
    echo "Access the UI at: https://localhost:8000"
    setup_auth0 "localhost:8000"
}

function launch_sensor {
    local k8s_dir="$1"

    local common_params="{ \"params\" : { \"namespace\": \"stackrox\" } }"

    local extra_config=""
    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
        extra_config+='"monitoringEndpoint": "monitoring.stackrox:443", '
    fi
    extra_config+="\"kubernetes\": $common_params }"

    get_cluster_zip localhost:8000 "$CLUSTER" KUBERNETES_CLUSTER "$MAIN_IMAGE" "$CLUSTER_API_ENDPOINT" "$k8s_dir" "$RUNTIME_SUPPORT" "$extra_config"

    echo "Deploying Sensor..."
    local unzip_dir="$k8s_dir/sensor-deploy/"
    rm -rf "$unzip_dir"
    unzip "$k8s_dir/sensor-deploy.zip" -d "$unzip_dir"
    $unzip_dir/sensor.sh
    echo

}
