#!/usr/bin/env bash

function launch_service {
    local dir="$1"
    local service="$2"

    if [[ "${OUTPUT_FORMAT}" == "helm" ]]; then
        for i in {1..5}; do
            if helm install "$dir/$service" --name $service --tiller-connection-timeout 10; then
                break
            fi
            sleep 5
            echo "Waiting for helm to respond"
        done
    else
        ${ORCH_CMD} create -R -f "$dir/$service"
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

    local unzip_dir="${k8s_dir}/central-deploy/"
    rm -rf "${unzip_dir}"
    if [[ -x "$(command -v roxctl)" && "$(roxctl version)" == "$MAIN_IMAGE_TAG" ]]; then
       rm -rf central-bundle "${k8s_dir}/central-bundle"
       roxctl central generate ${ORCH} ${EXTRA_ARGS[@]} --output-dir="central-bundle" --output-format="${OUTPUT_FORMAT}" --monitoring-password=stackrox \
           -i "${MAIN_IMAGE}" --monitoring-persistence-type="${STORAGE}" "${STORAGE}"
       cp -R central-bundle/ "${unzip_dir}/"
       rm -rf central-bundle
    else
       docker run --rm --env-file <(env | grep '^ROX_') "$MAIN_IMAGE" central generate ${ORCH} ${EXTRA_ARGS[@]} --output-format="${OUTPUT_FORMAT}" \
        --monitoring-password=stackrox -i "${MAIN_IMAGE}" --monitoring-persistence-type="${STORAGE}" "${STORAGE}" > "${k8s_dir}/central.zip"
        unzip "${k8s_dir}/central.zip" -d "${unzip_dir}"
    fi

    echo

    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
        echo "Deploying Monitoring..."
        $unzip_dir/monitoring/scripts/setup.sh
        launch_service $unzip_dir monitoring
        echo

        ${ORCH_CMD} -n stackrox patch deployment monitoring --patch "$(cat $k8s_dir/monitoring-resources-patch.yaml)"
    fi

	if [[ -f "${unzip_dir}/password" ]]; then
		export ROX_ADMIN_USER=admin
		export ROX_ADMIN_PASSWORD="$(< "${unzip_dir}/password")"
	fi

    echo "Deploying Central..."
    $unzip_dir/central/scripts/setup.sh
    launch_service $unzip_dir central
    echo

    # if we have specified that we want to use a load balancer, then use that endpoint instead of localhost
    if [[ "${LOAD_BALANCER}" == "lb" ]]; then
        # wait for LB
        echo "Waiting for LB to provision"
        LB_IP=""
        until [ -n "${LB_IP}" ]; do
            echo -n "."
            sleep 1
            LB_IP=$(kubectl -n stackrox get svc/central-loadbalancer -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
        done
        export API_ENDPOINT="${LB_IP}:443"
    else
        $unzip_dir/central/scripts/port-forward.sh 8000
    fi

    wait_for_central "${API_ENDPOINT}"
    echo "Successfully deployed Central!"
    echo "Access the UI at: https://${API_ENDPOINT}"
    setup_auth0 "${API_ENDPOINT}"
}

function launch_sensor {
    local k8s_dir="$1"

    local extra_config=()
    local extra_json_config=()
    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
        extra_config+=("--monitoring-endpoint=monitoring.stackrox:443")
        extra_json_config+=', "monitoringEndpoint": "monitoring.stackrox:443"'
    fi

    # Delete path
    rm -rf "$k8s_dir/sensor-deploy"

    if [[ "${ORCH}" == "k8s" && -x "$(command -v roxctl)" && "$(roxctl version)" == "$MAIN_IMAGE_TAG" ]]; then
        [[ -n "${ROX_ADMIN_PASSWORD}" ]] || { echo >&2 "ROX_ADMIN_PASSWORD not found! Cannot launch sensor."; return 1; }
        roxctl -p ${ROX_ADMIN_PASSWORD} --endpoint "${API_ENDPOINT}" sensor generate --image="${MAIN_IMAGE}" --central="$CLUSTER_API_ENDPOINT" --name="$CLUSTER" \
             --runtime="$RUNTIME_SUPPORT" --admission-controller="$ADMISSION_CONTROLLER" "${extra_config[@]+"${extra_config[@]}"}" ${ORCH}
        mv "sensor-${CLUSTER}" "$k8s_dir/sensor-deploy"
    else
        get_cluster_zip "$API_ENDPOINT" "$CLUSTER" ${CLUSTER_TYPE} "$MAIN_IMAGE" "$CLUSTER_API_ENDPOINT" "$k8s_dir" "$RUNTIME_SUPPORT" "$extra_json_config"
        unzip "$k8s_dir/sensor-deploy.zip" -d "$k8s_dir/sensor-deploy"
        rm "$k8s_dir/sensor-deploy.zip"
    fi

    echo "Deploying Sensor..."
    $k8s_dir/sensor-deploy/sensor.sh
    echo
}
