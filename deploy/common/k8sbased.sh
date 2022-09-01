#!/usr/bin/env bash

function realpath {
	[[ -n "$1" ]] || return 0
	python3 -c 'import os, sys; print(os.path.realpath(sys.argv[1]))' "$1"
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

  echo
  echo "**********"
  echo "$binary_name binary is being hot reloaded"
  echo "**********"
  echo

  binary_path=$(realpath "$(git rev-parse --show-toplevel)/bin/linux/${local_name}")
  kubectl -n stackrox patch "deploy/${deployment}" -p '{"spec":{"template":{"spec":{"containers":[{"name":"'${deployment}'","volumeMounts":[{"mountPath":"/stackrox/'${binary_name}'","name":"'binary-${local_name}'"}]}],"volumes":[{"hostPath":{"path":"'${binary_path}'","type":""},"name":"'binary-${local_name}'"}]}}}}'
  kubectl -n stackrox set env "deploy/$deployment" "ROX_HOTRELOAD=true"
}

function verify_orch {
    if [ "$ORCH" == "openshift" ]; then
        if kubectl api-versions | grep -q openshift.io; then
            return
        fi
        echo "Cannot find openshift orchestrator. Please check your kubeconfig for: $(kubectl config current-context)"
        exit 1
    fi
    if [ "$ORCH" == "k8s" ]; then
        if kubectl api-versions | grep -q configs.operator.openshift.io; then
            echo "Are you running an OpenShift orchestrator? Please use deploy/openshift/deploy*.sh to deploy."
            exit 1
        fi
        return
    fi
    echo "Unexpected orchestrator: $ORCH"
    exit 1
}

function local_dev {
      is_local_dev="false"
      if [[ $(kubectl get nodes -o json | jq '.items | length') == 1 ]]; then
        is_local_dev="true"
      fi
      echo "${is_local_dev}"
}

function launch_central {
    local k8s_dir="$1"
    local common_dir="${k8s_dir}/../common"

    verify_orch

    echo "Generating central config..."

    local EXTRA_ARGS=()
    local EXTRA_DOCKER_ARGS=()
    local STORAGE_ARGS=()

    local use_docker=1
    if [[ "${USE_LOCAL_ROXCTL:-}" == "true" ]]; then
      echo "Using $(command -v roxctl) for install due to USE_LOCAL_ROXCTL==true"
      use_docker=0
    elif [[ -x "$(command -v roxctl)" && "$(roxctl version)" == "$MAIN_IMAGE_TAG" ]]; then
      echo "Using $(command -v roxctl) for install due to version match with MAIN_IMAGE_TAG $MAIN_IMAGE_TAG"
      use_docker=0
    fi

    local DOCKER_PLATFORM_ARGS=()
    if [[ "$(uname -s)" == "Darwin" && "$(uname -m)" == "arm64" ]]; then
      DOCKER_PLATFORM_ARGS=("--platform linux/x86_64")
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

    if [ -n "${OUTPUT_FORMAT}" ]; then
        add_args "--output-format=${OUTPUT_FORMAT}"
    fi

    add_args "--lb-type=$LOAD_BALANCER"

    add_args "--offline=$OFFLINE_MODE"

    if [[ -n "$ROX_LICENSE_KEY" ]]; then
      add_args "--license"
      add_maybe_file_arg "${ROX_LICENSE_KEY}"
    fi

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

    if [[ "${ROX_POSTGRES_DATASTORE}" == "true" && -n "${CENTRAL_DB_IMAGE}" ]]; then
        add_args "--central-db-image=${CENTRAL_DB_IMAGE}"
    fi

    add_args "--image-defaults=${ROXCTL_ROX_IMAGE_FLAVOR}"

    pkill -f kubectl'.*port-forward.*' || true    # terminate stale port forwarding from earlier runs
    pkill -9 -f kubectl'.*port-forward.*' || true
    command -v oc >/dev/null && pkill -f oc'.*port-forward.*' || true    # terminate stale port forwarding from earlier runs
    command -v oc >/dev/null && pkill -9 -f oc'.*port-forward.*' || true

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

    if [[ "$POD_SECURITY_POLICIES" == "true" ]]; then
      add_args "--enable-pod-security-policies"
    fi

    local unzip_dir="${k8s_dir}/central-deploy/"
    rm -rf "${unzip_dir}"
    if ! (( use_docker )); then
        rm -rf central-bundle "${k8s_dir}/central-bundle"
        roxctl central generate "${ORCH}" "${EXTRA_ARGS[@]}" --output-dir="central-bundle" "${STORAGE}" "${STORAGE_ARGS[@]}"
        cp -R central-bundle/ "${unzip_dir}/"
        rm -rf central-bundle
    else
        docker run --rm ${DOCKER_PLATFORM_ARGS[@]} "${EXTRA_DOCKER_ARGS[@]}" --env-file <(env | grep '^ROX_') "$ROXCTL_IMAGE" \
        	central generate "${ORCH}" "${EXTRA_ARGS[@]}" "${STORAGE}" "${STORAGE_ARGS[@]}" > "${k8s_dir}/central.zip"
        unzip "${k8s_dir}/central.zip" -d "${unzip_dir}"
    fi

    echo
    if [[ -n "${TRUSTED_CA_FILE}" ]]; then
        if [[ -x "${unzip_dir}/scripts/ca-setup.sh" ]]; then
          "${unzip_dir}/scripts/ca-setup.sh" -f "${TRUSTED_CA_FILE}"
        else
          "${unzip_dir}/central/scripts/ca-setup.sh" -f "${TRUSTED_CA_FILE}"
        fi
    fi


    # Do not default to running monitoring locally for resource reasons, which can be overridden
    # with MONITORING_SUPPORT=true, otherwise default it to true on all other systems
    is_local_dev=$(local_dev)
    needs_monitoring="false"
    if [[ "$MONITORING_SUPPORT" == "true" || ( "${is_local_dev}" != "true" && -z "$MONITORING_SUPPORT" ) ]]; then
      needs_monitoring="true"
    fi
    if [[ "${needs_monitoring}" == "true" ]]; then
        echo "Deploying Monitoring..."
        helm_args=(
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

    ${KUBE_COMMAND:-kubectl} get namespace "${STACKROX_NAMESPACE}" &>/dev/null || \
      ${KUBE_COMMAND:-kubectl} create namespace "${STACKROX_NAMESPACE}"

    if [[ -f "$unzip_dir/values-public.yaml" ]]; then
      if [[ -n "${REGISTRY_USERNAME}" ]]; then
        $unzip_dir/scripts/setup.sh
      fi
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

      if [[ "$POD_SECURITY_POLICIES" == "true" ]]; then
        helm_args+=(
          --set system.enablePodSecurityPolicies=true
        )
      fi

      if [[ "$ROX_MANAGED_CENTRAL" == "true" ]]; then
        helm_args+=(
          --set customize.central.envVars.ROX_MANAGED_CENTRAL="${ROX_MANAGED_CENTRAL}"
        )
      fi

      if [[ -n "$CI" ]]; then
        helm lint "$unzip_dir/chart"
        helm lint "$unzip_dir/chart" -n stackrox
        helm lint "$unzip_dir/chart" -n stackrox "${helm_args[@]}"
      fi
      helm install -n stackrox stackrox-central-services "$unzip_dir/chart" \
          "${helm_args[@]}"
    else
      if [[ -n "${REGISTRY_USERNAME}" ]]; then
        $unzip_dir/central/scripts/setup.sh
      fi
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

      if [[ "$ROX_MANAGED_CENTRAL" == "true" ]]; then
        ${ORCH_CMD} -n stackrox set env deploy/central ROX_MANAGED_CENTRAL="${ROX_MANAGED_CENTRAL}"
      fi

      if [[ "$SCANNER_SUPPORT" == "true" ]]; then
          echo "Deploying Scanner..."
          if [[ -n "${REGISTRY_USERNAME}" ]]; then
            $unzip_dir/scanner/scripts/setup.sh
          fi
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

    if [[ "${is_local_dev}" == "true" && "${ROX_HOTRELOAD}" == "true" ]]; then
      hotload_binary central central central
    fi

    # Wait for any pending changes to Central deployment to get reconciled before trying to connect it.
    # On some systems there's a race condition when port-forward connects to central but its pod then gets deleted due
    # to ongoing modifications to the central deployment. This port-forward dies and the script hangs "Waiting for
    # Central to respond" until it times out. Waiting for rollout status should help not get into such situation.
    rollout_wait_timeout="3m"
    if [[ "${IS_RACE_BUILD:-}" == "true" ]]; then
      rollout_wait_timeout="9m"
    fi
    kubectl -n stackrox rollout status deploy/central --timeout="$rollout_wait_timeout"

    # if we have specified that we want to use a load balancer, then use that endpoint instead of localhost
    if [[ "${LOAD_BALANCER}" == "lb" ]]; then
        # wait for LB
        echo "Waiting for LB to provision"
        LB_IP=""
        until [[ -n "${LB_IP}" ]]; do
            echo -n "."
            sleep 1
            LB_IP=$(kubectl -n stackrox get svc/central-loadbalancer -o json | jq -r '.status.loadBalancer.ingress[0] | .ip // .hostname')
            if [[ "$LB_IP" == "null" ]]; then
              unset LB_IP
            fi
        done
        export API_ENDPOINT="${LB_IP}:443"
        echo
        echo "API_ENDPOINT set to [$API_ENDPOINT]"
    elif [[ "${LOAD_BALANCER}" == "route" ]]; then
        # wait for route
        echo "Waiting for route to provision"
        ROUTE_HOST=""
        until [ -n "${ROUTE_HOST}" ]; do
            echo -n "."
            sleep 1
            ROUTE_HOST=$(kubectl -n stackrox get route/central -o jsonpath='{.status.ingress[0].host}')
        done
        export API_ENDPOINT="${ROUTE_HOST}:443"
    else
        $central_scripts_dir/port-forward.sh 8000
    fi

    if [[ "${needs_monitoring}" == "true" ]]; then
      "${COMMON_DIR}/monitoring.sh"
    fi

    if [[ -n "$CI" ]]; then
        # Needed for GKE and OpenShift clusters
        echo "Sleep for 2 minutes to allow for stabilization"
        sleep 120
    fi

    wait_for_central "${API_ENDPOINT}"
    echo "Successfully deployed Central!"

    echo "Access the UI at: https://${API_ENDPOINT}"
    if [[ "${ROX_DEV_AUTH0_CLIENT_SECRET}" != "" ]]; then
        setup_auth0 "${API_ENDPOINT}" "${ROX_DEV_AUTH0_CLIENT_SECRET}"
    fi
}

function launch_sensor {
    local k8s_dir="$1"
    local common_dir="${k8s_dir}/../common"

    local extra_config=()
    local extra_json_config=''
    local extra_helm_config=()

    verify_orch

    if [[ "$ADMISSION_CONTROLLER" == "true" ]]; then
      extra_config+=("--admission-controller-listen-on-creates=true")
    	extra_json_config+=', "admissionController": true'
    	extra_helm_config+=(--set "admissionControl.listenOnCreates=true")
    fi
    if [[ "$ADMISSION_CONTROLLER_UPDATES" == "true" ]]; then
    	extra_config+=("--admission-controller-listen-on-updates=true")
    	extra_json_config+=', "admissionControllerUpdates": true'
    	extra_helm_config+=(--set "admissionControl.listenOnUpdates=true")
    fi
    if [[ -n "$ADMISSION_CONTROLLER_POD_EVENTS" ]]; then
      local bool_val
      bool_val="$(echo "$ADMISSION_CONTROLLER_POD_EVENTS" | tr '[:upper:]' '[:lower:]')"
      if [[ "$bool_val" != "true" ]]; then
        bool_val="false"
      fi
      extra_config+=("--admission-controller-listen-on-events=${bool_val}")
    	extra_json_config+=", \"admissionControllerEvents\": ${bool_val}"
    	extra_helm_config+=(--set "admissionControl.listenOnEvents=${bool_val}")
    fi

    if [[ -n "$COLLECTOR_IMAGE_REPO" ]]; then
        extra_config+=("--collector-image-repository=${COLLECTOR_IMAGE_REPO}")
        extra_json_config+=", \"collectorImage\": \"${COLLECTOR_IMAGE_REPO}\""
        extra_helm_config+=(--set "image.collector.repository=${COLLECTOR_IMAGE_REPO}")
    fi

    if [[ -n "$ROXCTL_TIMEOUT" ]]; then
      echo "Extending roxctl timeout to $ROXCTL_TIMEOUT"
      extra_config+=("--timeout=$ROXCTL_TIMEOUT")
    fi

    # Delete path
    rm -rf "$k8s_dir/sensor-deploy"

    if [[ -z "$CI" && -z "${SENSOR_HELM_DEPLOY:-}" && -x "$(command -v helm)" && "$(helm version --short)" == v3.* ]]; then
      echo >&2 "================================================================================================"
      echo >&2 "NOTE: Based on your environment, you are using the Helm-based deployment method."
      echo >&2 "      To disable the Helm based installation set SENSOR_HELM_DEPLOY=false"
      echo >&2 "================================================================================================"
      SENSOR_HELM_DEPLOY=true
    fi

    if [[ "${SENSOR_HELM_DEPLOY:-}" == "true" ]]; then
      local sensor_namespace="${SENSOR_HELM_OVERRIDE_NAMESPACE:-stackrox}"
      mkdir "$k8s_dir/sensor-deploy"
      touch "$k8s_dir/sensor-deploy/init-bundle.yaml"
      chmod 0600 "$k8s_dir/sensor-deploy/init-bundle.yaml"
      curl_central "https://${API_ENDPOINT}/v1/cluster-init/init-bundles" \
          -XPOST -d '{"name":"deploy-'"${CLUSTER}-$(date '+%Y%m%d%H%M%S')"'"}' \
          | jq '.helmValuesBundle' -r | base64 --decode >"$k8s_dir/sensor-deploy/init-bundle.yaml"

      curl_central "https://${API_ENDPOINT}/api/extensions/helm-charts/secured-cluster-services.zip" \
          -o "$k8s_dir/sensor-deploy/chart.zip"
      mkdir "$k8s_dir/sensor-deploy/chart"
      unzip "$k8s_dir/sensor-deploy/chart.zip" -d "$k8s_dir/sensor-deploy/chart"

      helm_args=(
        -f "$k8s_dir/sensor-deploy/init-bundle.yaml"
        --set "imagePullSecrets.allowNone=true"
        --set "clusterName=${CLUSTER}"
        --set "centralEndpoint=${CLUSTER_API_ENDPOINT}"
        --set "image.main.repository=${MAIN_IMAGE_REPO}"
        --set "collector.collectionMethod=$(echo "$COLLECTION_METHOD" | tr '[:lower:]' '[:upper:]')"
        --set "env.openshift=$([[ "$ORCH" == "openshift" ]] && echo "true" || echo "false")"
      )
      if [[ -f "$k8s_dir/sensor-deploy/chart/feature-flag-values.yaml" ]]; then
        helm_args+=(-f "$k8s_dir/sensor-deploy/chart/feature-flag-values.yaml")
      fi
      if [[ "$sensor_namespace" != "stackrox" ]]; then
        helm_args+=(--set "allowNonstandardNamespace=true")
      fi

      if [[ "$SENSOR_HELM_MANAGED" == "true" ]]; then
        helm_args+=(--set "helmManaged=true")
      else
        helm_args+=(--set "helmManaged=false")
      fi

      if [[ -n "$CI" ]]; then
        helm lint "$k8s_dir/sensor-deploy/chart"
        helm lint "$k8s_dir/sensor-deploy/chart" -n stackrox
        helm lint "$k8s_dir/sensor-deploy/chart" -n stackrox "${helm_args[@]}" "${extra_helm_config[@]}"
      fi
      if [[ "$sensor_namespace" != "stackrox" ]]; then
        kubectl create namespace "$sensor_namespace" &>/dev/null || true
        kubectl -n "$sensor_namespace" get secret stackrox &>/dev/null || kubectl -n "$sensor_namespace" create -f - < <("${common_dir}/pull-secret.sh" stackrox docker.io)
      fi
      helm upgrade --install -n "$sensor_namespace" --create-namespace stackrox-secured-cluster-services "$k8s_dir/sensor-deploy/chart" \
          "${helm_args[@]}" "${extra_helm_config[@]}"
    else
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

      namespace=stackrox
      if [[ -n "$NAMESPACE_OVERRIDE" ]]; then
        namespace="$NAMESPACE_OVERRIDE"
        echo "Changing namespace to $NAMESPACE_OVERRIDE"
        ls $k8s_dir/sensor-deploy/*.yaml | while read file; do sed -i'.original' -e 's/namespace: stackrox/namespace: '"$NAMESPACE_OVERRIDE"'/g' $file; done
        sed -itmp.bak 's/set -e//g' $k8s_dir/sensor-deploy/sensor.sh
      fi

      echo "Deploying Sensor..."
      NAMESPACE="$namespace" $k8s_dir/sensor-deploy/sensor.sh
    fi

    if [[ -n "${ROX_AFTERGLOW_PERIOD}" ]]; then
       kubectl -n stackrox set env ds/collector ROX_AFTERGLOW_PERIOD="${ROX_AFTERGLOW_PERIOD}"
    fi

    if [[ -n "${CI}" || $(kubectl get nodes -o json | jq '.items | length') == 1 ]]; then
       if [[ "${ROX_HOTRELOAD}" == "true" ]]; then
         hotload_binary bin/kubernetes-sensor kubernetes sensor
       fi
       if [[ -z "${IS_RACE_BUILD}" ]]; then
           kubectl -n stackrox patch deploy/sensor --patch '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"limits":{"cpu":"500m","memory":"500Mi"},"requests":{"cpu":"500m","memory":"500Mi"}}}]}}}}'
       fi
    fi
    if [[ "$MONITORING_SUPPORT" == "true" || ( "$(local_dev)" != "true" && -z "$MONITORING_SUPPORT" ) ]]; then
      "${COMMON_DIR}/monitoring.sh"
    fi

    echo
}
