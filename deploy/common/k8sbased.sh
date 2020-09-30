#!/usr/bin/env bash

function realpath {
	[[ -n "$1" ]] || return 0
	python -c 'import os, sys; print(os.path.realpath(sys.argv[1]))' "$1"
}

function launch_service {
    local dir="$1"
    local service="$2"

    if [[ "${OUTPUT_FORMAT}" == "helm" ]]; then
        local helm_version
        helm_version="$(helm version --short -c | sed -e 's/^Client: //g')"
        if [[ -z "$helm_version" ]]; then
          echo >&2 "helm not found or doesn't work"
          exit 1
        elif [[ "$helm_version" == v2.* ]]; then
          echo "Detected Helm v2"
          if [[ -f "$dir/values-public.yaml" ]]; then
            echo "The new helm chart cannot be deployed with Helm ${helm_version}."
            echo "Please upgrade to at least Helm v3.1.0"
            return 1
          fi
          helm_install() { helm install "$dir/$1" --name "$1" --tiller-connection-timeout 10 ; }
        elif [[ "$helm_version" == v3.* ]]; then
          echo "Detected Helm v3"
          helm_install() { helm install "$1" "$dir/$1" ; }
        else
          echo "Unknown helm version: ${helm_version}"
          return 1
        fi

        for _ in {1..5}; do
            if helm_install "$service"; then
                break
            fi
            sleep 5
            echo "Waiting for helm to respond"
        done
    else
        ${ORCH_CMD} create -R -f "$dir/$service"
    fi
}

function hotload_binary {
  local binary_name="$1"
  local local_name="$2"
  local deployment="$3"

  binary_path=$(realpath "$(git rev-parse --show-toplevel)/bin/linux/${local_name}")
  kubectl -n stackrox patch "deploy/${deployment}" -p '{"spec":{"template":{"spec":{"containers":[{"name":"'${deployment}'","volumeMounts":[{"mountPath":"/stackrox/'${binary_name}'","name":"binary"}]}],"volumes":[{"hostPath":{"path":"'${binary_path}'","type":""},"name":"binary"}]}}}}'
}

function launch_central {
    local k8s_dir="$1"
    local common_dir="${k8s_dir}/../common"

    echo "Generating central config..."

    local EXTRA_ARGS=()
    local EXTRA_DOCKER_ARGS=()
    local STORAGE_ARGS=()

	local use_docker=1
    if [[ -x "$(command -v roxctl)" && "$(roxctl version)" == "$MAIN_IMAGE_TAG" ]]; then
    	use_docker=0
    fi

    add_args() {
    	EXTRA_ARGS+=("$@")
    }
    add_storage_args() {
        STORAGE_ARGS+=("$@")
    }
    add_maybe_file_arg() {
    	if [[ -f "$1" ]]; then
    		add_file_arg "$1"
    	else
    		add_args "$1"
    	fi
    }
    add_file_arg() {
    	if (( use_docker )); then
    		EXTRA_DOCKER_ARGS+=(-v "$(realpath "$1"):$(realpath "$1")")
    	fi
    	EXTRA_ARGS+=("$(realpath "$1")")
    }

    is_local_dev="false"
    if [[ $(kubectl get nodes -o json | jq '.items | length') == 1 ]]; then
      is_local_dev="true"
      echo "Running in local dev mode. Will patch resources down"
    fi

    if [ -n "${OUTPUT_FORMAT}" ]; then
        add_args "--output-format=${OUTPUT_FORMAT}"
    fi

    add_args "--lb-type=$LOAD_BALANCER"

    add_args "--offline=$OFFLINE_MODE"

    add_args "--license"
    add_maybe_file_arg "${ROX_LICENSE_KEY:-${common_dir}/dev-license.lic}"

    if [[ -n "$SCANNER_IMAGE" ]]; then
        add_args "--scanner-image=$SCANNER_IMAGE"
    fi

    if [[ -n "$SCANNER_DB_IMAGE" ]]; then
        add_args "--scanner-db-image=${SCANNER_DB_IMAGE}"
    fi

    if [[ -n "$ROX_DEFAULT_TLS_CERT_FILE" ]]; then
    	add_args "--default-tls-cert"
    	add_file_arg "$ROX_DEFAULT_TLS_CERT_FILE"
    	add_args "--default-tls-key"
    	add_file_arg "$ROX_DEFAULT_TLS_KEY_FILE"
    fi

    add_args -i "${MAIN_IMAGE}"

    pkill -f "$ORCH_CMD"'.*port-forward.*' || true    # terminate stale port forwarding from earlier runs
    pkill -9 -f "$ORCH_CMD"'.*port-forward.*' || true

    if [[ "${STORAGE_CLASS}" == "faster" ]]; then
        kubectl apply -f "${common_dir}/ssd-storageclass.yaml"
    fi

    if [[ "${STORAGE}" == "none" && -n $STORAGE_CLASS ]]; then
        echo "Invalid deploy script config. STORAGE is set to none, but STORAGE_CLASS is set"
        exit 1
    fi

    if [[ -n $STORAGE_CLASS ]]; then
        add_storage_args "--storage-class=$STORAGE_CLASS"
    fi

    if [[ "${STORAGE}" == "pvc" && -n "${STORAGE_SIZE}" ]]; then
	      add_storage_args "--size=${STORAGE_SIZE}"
    fi

    if [[ -n "${ROXDEPLOY_CONFIG_FILE_MAP}" ]]; then
    	add_args "--with-config-file=${ROXDEPLOY_CONFIG_FILE_MAP}"
    fi

    local unzip_dir="${k8s_dir}/central-deploy/"
    rm -rf "${unzip_dir}"
    if ! (( use_docker )); then
        rm -rf central-bundle "${k8s_dir}/central-bundle"
        roxctl central generate "${ORCH}" "${EXTRA_ARGS[@]}" --output-dir="central-bundle" "${STORAGE}" "${STORAGE_ARGS[@]}"
        cp -R central-bundle/ "${unzip_dir}/"
        rm -rf central-bundle
    else
        docker run --rm "${EXTRA_DOCKER_ARGS[@]}" --env-file <(env | grep '^ROX_') "$MAIN_IMAGE" \
        	central generate "${ORCH}" "${EXTRA_ARGS[@]}" "${STORAGE}" "${STORAGE_ARGS[@]}" > "${k8s_dir}/central.zip"
        unzip "${k8s_dir}/central.zip" -d "${unzip_dir}"
    fi

    echo
    if [[ -n "${TRUSTED_CA_FILE}" ]]; then
        "${unzip_dir}/central/scripts/ca-setup.sh" -f "${TRUSTED_CA_FILE}"
    fi

    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
        echo "Deploying Monitoring..."
        helm_args=(
          -f "${COMMON_DIR}/monitoring-values.yaml"
          --set image="${MONITORING_IMAGE}"
          --set persistence.type="${STORAGE}"
          --set exposure.type="${MONITORING_LOAD_BALANCER}"
        )
        if [[ "${is_local_dev}" == "true" ]]; then
          helm_args+=(-f "${COMMON_DIR}/monitoring-values-local.yaml")
        fi

        helm install -n stackrox --create-namespace stackrox-monitoring "${COMMON_DIR}/../charts/monitoring" "${helm_args[@]}"
        echo
    fi

	if [[ -f "${unzip_dir}/password" ]]; then
		export ROX_ADMIN_USER=admin
		export ROX_ADMIN_PASSWORD="$(< "${unzip_dir}/password")"
	fi

    echo "Deploying Central..."

    if [[ -f "$unzip_dir/values-public.yaml" ]]; then
      $unzip_dir/scripts/setup.sh
      central_scripts_dir="$unzip_dir/scripts"

      # New helm setup flavor
      helm_args=(
        -f "$unzip_dir/values-public.yaml"
        -f "$unzip_dir/values-private.yaml"
        --set-string imagePullSecrets.useExisting="stackrox;stackrox-scanner"
      )

      if [[ "$SCANNER_SUPPORT" != "true" ]]; then
        helm_args+=(--set scanner.disable=true)
      fi

      if [[ "${is_local_dev}" == "true" ]]; then
        helm_args+=(-f "${COMMON_DIR}/local-dev-values.yaml")
      elif [[ -n "$CI" ]]; then
        helm_args+=(-f "${COMMON_DIR}/ci-values.yaml")
      fi

      if [[ "${CGO_CHECKS}" == "true" ]]; then
        echo "CGO_CHECKS set to true. Setting GODEBUG=cgocheck=2 and MUTEX_WATCHDOG_TIMEOUT_SECS=15"
        # Extend mutex watchdog timeout because cgochecks hamper performance
        helm_args+=(
          --set customize.central.envVars.GODEBUG=cgocheck=2
          --set customize.central.envVars.MUTEX_WATCHDOG_TIMEOUT_SECS=15
        )
      fi

      # set logging options
      if [[ -n $LOGLEVEL ]]; then
        helm_args+=(
          --set customize.central.envVars.LOGLEVEL="${LOGLEVEL}"
        )
      fi
      if [[ -n $MODULE_LOGLEVELS ]]; then
        helm_args+=(
          --set customize.central.envVars.MODULE_LOGLEVELS="${MODULE_LOGLEVELS}"
        )
      fi

      helm install -n stackrox stackrox-central-services "$unzip_dir/chart" \
          "${helm_args[@]}"
    else
      $unzip_dir/central/scripts/setup.sh
      central_scripts_dir="$unzip_dir/central/scripts"
      launch_service $unzip_dir central
      echo

      if [[ "${is_local_dev}" == "true" ]]; then
          kubectl -n stackrox patch deploy/central --patch '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"limits":{"cpu":"1","memory":"4Gi"},"requests":{"cpu":"1","memory":"1Gi"}}}]}}}}'
      fi

      if [[ "${CGO_CHECKS}" == "true" ]]; then
        echo "CGO_CHECKS set to true. Setting GODEBUG=cgocheck=2 and MUTEX_WATCHDOG_TIMEOUT_SECS=15"
        # Extend mutex watchdog timeout because cgochecks hamper performance
        ${ORCH_CMD} -n stackrox set env deploy/central GODEBUG=cgocheck=2 MUTEX_WATCHDOG_TIMEOUT_SECS=15
      fi

      # set logging options
      if [[ -n $LOGLEVEL ]]; then
        ${ORCH_CMD} -n stackrox set env deploy/central LOGLEVEL="${LOGLEVEL}"
      fi
      if [[ -n $MODULE_LOGLEVELS ]]; then
        ${ORCH_CMD} -n stackrox set env deploy/central MODULE_LOGLEVELS="${MODULE_LOGLEVELS}"
      fi

      if [[ "$SCANNER_SUPPORT" == "true" ]]; then
          echo "Deploying Scanning..."
          $unzip_dir/scanner/scripts/setup.sh
          launch_service $unzip_dir scanner

          if [[ -n "$CI" ]]; then
            ${ORCH_CMD} -n stackrox patch deployment scanner --patch "$(cat "${common_dir}/scanner-patch.yaml")"
            ${ORCH_CMD} -n stackrox patch hpa scanner --patch "$(cat "${common_dir}/scanner-hpa-patch.yaml")"
          elif [[ "${is_local_dev}" == "true" ]]; then
            ${ORCH_CMD} -n stackrox patch deployment scanner --patch "$(cat "${common_dir}/scanner-local-patch.yaml")"
            ${ORCH_CMD} -n stackrox patch hpa scanner --patch "$(cat "${common_dir}/scanner-hpa-patch.yaml")"
          fi
          echo
      fi
    fi

    if [[ "${is_local_dev}" == "true" && "${HOTRELOAD}" == "true" ]]; then
      hotload_binary central central central
    fi

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
        $central_scripts_dir/port-forward.sh 8000
    fi

    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
      "${COMMON_DIR}/monitoring.sh"
    fi

    if [[ -n "$CI" ]]; then
        echo "Sleep for 1 minute to allow for GKE stabilization"
        sleep 60
    fi

    wait_for_central "${API_ENDPOINT}"
    echo "Successfully deployed Central!"

    echo "Access the UI at: https://${API_ENDPOINT}"
    if [[ "$AUTH0_SUPPORT" == "true" ]]; then
        setup_auth0 "${API_ENDPOINT}"
    fi
}

function launch_sensor {
    local k8s_dir="$1"

    local extra_config=()
    local extra_json_config=()

    if [[ "$ADMISSION_CONTROLLER" == "true" ]]; then
        extra_config+=("--create-admission-controller=true")
    	extra_json_config+=', "admissionController": true'
    fi
    if [[ "$ADMISSION_CONTROLLER_UPDATES" == "true" ]]; then
    	extra_config+=("--admission-controller-listen-on-updates=true")
    	extra_json_config+=', "admissionControllerUpdates": true'
    fi

    if [[ -n "$COLLECTOR_IMAGE_REPO" ]]; then
        extra_config+=("--collector-image-repository=${COLLECTOR_IMAGE_REPO}")
    fi

    # Disabled this special-case for now.
    # Shall be completely removed when the ROX_SUPPORT_SLIM_COLLECTOR_MODE feature flag is removed.
    #
    # Let's see if CI works with slim images now (give that slim collector images are now being published to docker.io
    # and collector has its bootstrapping timeout increased).
    #
    # # For now the CI setup requires non-slim collectors.
    # IS_CENTRAL_ENABLED="$(get_central_feature_flag_enabled "$API_ENDPOINT" ROX_SUPPORT_SLIM_COLLECTOR_MODE)"
    # if [[ "${IS_CENTRAL_ENABLED}" == "true" ]]; then
    #     extra_config+=("--slim-collector=false")
    #     extra_json_config+=', "slimCollector": false'
    # fi

    # Delete path
    rm -rf "$k8s_dir/sensor-deploy"

    if [[ -x "$(command -v roxctl)" && "$(roxctl version)" == "$MAIN_IMAGE_TAG" ]]; then
        [[ -n "${ROX_ADMIN_PASSWORD}" ]] || { echo >&2 "ROX_ADMIN_PASSWORD not found! Cannot launch sensor."; return 1; }
        roxctl -p ${ROX_ADMIN_PASSWORD} --endpoint "${API_ENDPOINT}" sensor generate --main-image-repository="${MAIN_IMAGE_REPO}" --central="$CLUSTER_API_ENDPOINT" --name="$CLUSTER" \
             --collection-method="$COLLECTION_METHOD" \
             "${ORCH}" \
             "${extra_config[@]+"${extra_config[@]}"}"
        mv "sensor-${CLUSTER}" "$k8s_dir/sensor-deploy"
    else
        get_cluster_zip "$API_ENDPOINT" "$CLUSTER" ${CLUSTER_TYPE} "${MAIN_IMAGE_REPO}" "$CLUSTER_API_ENDPOINT" "$k8s_dir" "$COLLECTION_METHOD" "$extra_json_config"
        unzip "$k8s_dir/sensor-deploy.zip" -d "$k8s_dir/sensor-deploy"
        rm "$k8s_dir/sensor-deploy.zip"
    fi

    echo "Deploying Sensor..."
    $k8s_dir/sensor-deploy/sensor.sh

    if [[ -n "${CI}" || $(kubectl get nodes -o json | jq '.items | length') == 1 ]]; then
       if [[ "${HOTRELOAD}" == "true" ]]; then
         hotload_binary kubernetes-sensor kubernetes sensor
       fi
       if [[ -z "${IS_RACE_BUILD}" ]]; then
           kubectl -n stackrox patch deploy/sensor --patch '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"limits":{"cpu":"500m","memory":"500Mi"},"requests":{"cpu":"500m","memory":"500Mi"}}}]}}}}'
       fi
    fi

    if [[ "$MONITORING_SUPPORT" == "true" ]]; then
      "${COMMON_DIR}/monitoring.sh"
    fi

    echo
}
