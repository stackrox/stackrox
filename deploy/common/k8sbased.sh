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
          helm_install() { helm upgrade --install "$dir/$1" --name "$1" --tiller-connection-timeout 10 ; }
        elif [[ "$helm_version" == v3.* ]]; then
          echo "Detected Helm v3"
          helm_install() { helm upgrade --install "$1" "$dir/$1" ; }
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
        ${ORCH_CMD} apply -R -f "$dir/$service"
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

  binary_path=$(realpath "$(git rev-parse --show-toplevel)/bin/linux_amd64/${local_name}")
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

# Checks if central already exists in this cluster.
# If yes, the user is asked if they want to continue. If they answer no, then the script is terminated.
function prompt_if_central_exists() {
    if "${ORCH_CMD}" -n stackrox get deployment central 2>&1; then
        yes_no_prompt "Detected there is already a central running on this cluster. Are you sure you want to proceed with this deploy?" || { echo >&2 "Exiting as requested"; exit 1; }
    fi
}

# yes_no_prompt "<message>" displays the given message and prompts the user to
# input 'yes' or 'no'. The return value is 0 if the user has entered 'yes', 1
# if they answered 'no', and 2 if the read was aborted (^C/^D) or no valid
# answer was given after three tries.
function yes_no_prompt() {
  local prompt="$1"
  local tries=0
  [[ -z "$prompt" ]] || echo >&2 "$prompt"
  local answer=""
  while (( tries < 3 )) && { echo -n "Type 'yes' or 'no': "; read answer; } ; do
    answer="$(echo "$answer" | tr '[:upper:]' '[:lower:]')"
    [[ "$answer" == "yes" ]] && return 0
    [[ "$answer" == "no" ]] && return 1
    tries=$((tries + 1))
  done
  echo "Aborted."
  return 2
}

function launch_central {
    local k8s_dir="$1"
    local common_dir="${k8s_dir}/../common"

    verify_orch
    if [[ -z "$CI" ]]; then
        prompt_if_central_exists
    fi

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
        if [[ "${ROX_POSTGRES_DATASTORE}" == "true" ]]; then
            add_storage_args "--db-storage-class=$STORAGE_CLASS"
        fi
    fi

    if [[ "${STORAGE}" == "pvc" && -n "${STORAGE_SIZE}" ]]; then
	      add_storage_args "--size=${STORAGE_SIZE}"
        if [[ "${ROX_POSTGRES_DATASTORE}" == "true" ]]; then
            add_storage_args "--db-size=${STORAGE_SIZE}"
        fi
    fi

    if [[ -n "${ROXDEPLOY_CONFIG_FILE_MAP}" ]]; then
    	add_args "--with-config-file=${ROXDEPLOY_CONFIG_FILE_MAP}"
    fi

    SUPPORTS_PSP=$(kubectl api-resources | grep "podsecuritypolicies" -c || true)
    if [[ "${SUPPORTS_PSP}" -eq 0 ]]; then
        echo "Pod security policies are not supported on this cluster. Skipping..."
        POD_SECURITY_POLICIES="false"
    fi

    if [[ -n "$POD_SECURITY_POLICIES" ]]; then
      add_args "--enable-pod-security-policies=${POD_SECURITY_POLICIES}"
    fi

    add_args "--declarative-config-config-maps=declarative-configurations"
    add_args "--declarative-config-secrets=sensitive-declarative-configurations"

    if [[ -n "${ROX_TELEMETRY_STORAGE_KEY_V1}" ]]; then
      add_args "--enable-telemetry=true"
    else
      add_args "--enable-telemetry=false"
    fi

    if [[ -n "${ROX_OPENSHIFT_VERSION}" ]]; then
      add_args "--openshift-version=${ROX_OPENSHIFT_VERSION}"
    fi

    local unzip_dir="${k8s_dir}/central-deploy/"
    rm -rf "${unzip_dir}"
    if ! (( use_docker )); then
        rm -rf central-bundle "${k8s_dir}/central-bundle"
        roxctl central generate "${ORCH}" "${EXTRA_ARGS[@]}" --output-dir="central-bundle" "${STORAGE}" "${STORAGE_ARGS[@]}"
        cp -R central-bundle/ "${unzip_dir}/"
        rm -rf central-bundle
    else
        docker run --rm "${EXTRA_DOCKER_ARGS[@]}" --env-file <(env | grep '^ROX_') "$ROXCTL_IMAGE" \
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

        helm dependency update "${COMMON_DIR}/../charts/monitoring"
        envsubst < "${COMMON_DIR}/../charts/monitoring/values.yaml" > "${COMMON_DIR}/../charts/monitoring/values_substituted.yaml"
        helm upgrade -n stackrox --install --create-namespace stackrox-monitoring "${COMMON_DIR}/../charts/monitoring" --values "${COMMON_DIR}/../charts/monitoring/values_substituted.yaml" "${helm_args[@]}"
        rm "${COMMON_DIR}/../charts/monitoring/values_substituted.yaml"
        echo "Deployed Monitoring..."
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

      if [[ -n "$POD_SECURITY_POLICIES" ]]; then
        helm_args+=(
          --set system.enablePodSecurityPolicies="${POD_SECURITY_POLICIES}"
        )
      fi

      if [[ "$ROX_MANAGED_CENTRAL" == "true" ]]; then
        helm_args+=(
          --set env.managedServices=true
        )
      fi

      if [[ -n "$ROX_OPENSHIFT_VERSION" ]]; then
        helm_args+=(
          --set env.openshift="${ROX_OPENSHIFT_VERSION}"
        )
      fi

      if [[ "$ROX_SCANNER_V4_ENABLED" == "true" ]]; then
        helm_args+=(
          --set scannerV4.disable=false
        )
      fi

      local helm_chart="$unzip_dir/chart"

      if [[ -n "${CENTRAL_CHART_DIR_OVERRIDE}" ]]; then
        echo "Using override central helm chart from ${CENTRAL_CHART_DIR_OVERRIDE}"
        helm_chart="${CENTRAL_CHART_DIR_OVERRIDE}"
      fi

      if [[ -n "$CI" ]]; then
        helm lint "${helm_chart}"
        helm lint "${helm_chart}" -n stackrox
        helm lint "${helm_chart}" -n stackrox "${helm_args[@]}"
      fi

      helm upgrade --install -n stackrox stackrox-central-services "$helm_chart" \
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
          if [[ "${ROX_POSTGRES_DATASTORE}" == "true" ]]; then
            kubectl -n stackrox patch deploy/central-db --patch '{"spec":{"template":{"spec":{"initContainers":[{"name":"init-db","resources":{"limits":{"cpu":"1","memory":"4Gi"},"requests":{"cpu":1,"memory":"1Gi"}}}],"containers":[{"name":"central-db","resources":{"limits":{"cpu":"1","memory":"4Gi"},"requests":{"cpu":"1","memory":"1Gi"}}}]}}}}'
          fi
      elif [[ "${ROX_POSTGRES_DATASTORE}" == "true" ]]; then
          ${ORCH_CMD} -n stackrox patch deploy/central-db --patch "$(cat "${common_dir}/central-db-patch.yaml")"
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
        echo "ROX_MANAGED_CENTRAL=true is only supported in conjunction with OUTPUT_FORMAT=helm"
        exit 1
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

    if [[ -n "${ROX_DEV_INTERNAL_SSO_CLIENT_SECRET}" ]]; then
        ${KUBE_COMMAND:-kubectl} create secret generic sensitive-declarative-configurations -n "${STACKROX_NAMESPACE}" &>/dev/null
        setup_internal_sso "${API_ENDPOINT}" "${ROX_DEV_INTERNAL_SSO_CLIENT_SECRET}"
    fi

    if [[ "${is_local_dev}" == "true" && "${ROX_HOTRELOAD}" == "true" ]]; then
      hotload_binary central central central
    fi

    # Wait for any pending changes to Central deployment to get reconciled before trying to connect it.
    # On some systems there's a race condition when port-forward connects to central but its pod then gets deleted due
    # to ongoing modifications to the central deployment. This port-forward dies and the script hangs "Waiting for
    # Central to respond" until it times out. Waiting for rollout status should help not get into such situation.
    rollout_wait_timeout="4m"
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

    if [[ -n "$ROXCTL_TIMEOUT" ]]; then
      echo "Extending roxctl timeout to $ROXCTL_TIMEOUT"
      extra_config+=("--timeout=$ROXCTL_TIMEOUT")
    fi

    SUPPORTS_PSP=$(kubectl api-resources | grep "podsecuritypolicies" -c || true)
    if [[ "${SUPPORTS_PSP}" -eq 0 ]]; then
        echo "Pod security policies are not supported on this cluster. Skipping..."
        POD_SECURITY_POLICIES="false"
    fi

    if [[ -n "$POD_SECURITY_POLICIES" ]]; then
        extra_config+=("--enable-pod-security-policies=${POD_SECURITY_POLICIES}")
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
        --set "collector.collectionMethod=$(echo "$COLLECTION_METHOD" | tr '[:lower:]' '[:upper:]')"
      )
      if [[ -n "${ROX_OPENSHIFT_VERSION}" ]]; then
        helm_args+=(
          --set env.openshift="${ROX_OPENSHIFT_VERSION}"
        )
      else
        helm_args+=(
          --set "env.openshift=$([[ "$ORCH" == "openshift" ]] && echo "true" || echo "false")"
        )
      fi

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

      if [[ "$SENSOR_SCANNER_SUPPORT" == "true" ]]; then
        helm_args+=(--set scanner.disable=false)
      fi

      if [[ -n "$LOGLEVEL" ]]; then
        helm_args+=(
          --set customize.envVars.LOGLEVEL="${LOGLEVEL}"
        )
      fi

      local helm_chart="$k8s_dir/sensor-deploy/chart"

      if [[ -n "${SENSOR_CHART_DIR_OVERRIDE}" ]]; then
        echo "Using override sensor helm chart from ${SENSOR_CHART_DIR_OVERRIDE}"
        helm_chart="${SENSOR_CHART_DIR_OVERRIDE}"
      fi

      if [[ -n "$CI" ]]; then
        helm lint "${helm_chart}"
        helm lint "${helm_chart}" -n stackrox
        helm lint "${helm_chart}" -n stackrox "${helm_args[@]}" "${extra_helm_config[@]}"
      fi

      if [[ "$sensor_namespace" != "stackrox" ]]; then
        kubectl create namespace "$sensor_namespace" &>/dev/null || true
        kubectl -n "$sensor_namespace" get secret stackrox &>/dev/null || kubectl -n "$sensor_namespace" create -f - < <("${common_dir}/pull-secret.sh" stackrox docker.io)
      fi

      helm upgrade --install -n "$sensor_namespace" --create-namespace stackrox-secured-cluster-services "$helm_chart" \
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

    # For local installations (e.g. on Colima): hotload binary and update resource requests
    if [[ "$(local_dev)" == "true" ]]; then
        if [[ "${ROX_HOTRELOAD}" == "true" ]]; then
            hotload_binary bin/kubernetes-sensor kubernetes sensor
        fi
        if [[ -z "${IS_RACE_BUILD}" ]]; then
           kubectl -n stackrox patch deploy/sensor --patch '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"limits":{"cpu":"500m","memory":"500Mi"},"requests":{"cpu":"500m","memory":"500Mi"}}}]}}}}'
        fi
    fi

    # When running CI steps or when SENSOR_DEV_RESOURCES is set to true: only update resource requests
    if [[ -n "${CI}" || "${SENSOR_DEV_RESOURCES}" == "true" ]]; then
        if [[ -z "${IS_RACE_BUILD}" ]]; then
            kubectl -n stackrox patch deploy/sensor --patch '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"limits":{"cpu":"500m","memory":"500Mi"},"requests":{"cpu":"500m","memory":"500Mi"}}}]}}}}'
        fi
    fi

    if [[ "$MONITORING_SUPPORT" == "true" || ( "$(local_dev)" != "true" && -z "$MONITORING_SUPPORT" ) ]]; then
      "${COMMON_DIR}/monitoring.sh"
    fi

    # If deploying with chaos proxy enabled, patch sensor to add toxiproxy proxy deployment
    if [[ "$CHAOS_PROXY" == "true" ]]; then
        original_endpoint=$(kubectl -n stackrox get deploy/sensor -ojsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ROX_CENTRAL_ENDPOINT")].value}')

        echo "Patching sensor with toxiproxy container"
        kubectl -n stackrox patch deploy/sensor --type=json -p="$(cat "${common_dir}/sensor-toxiproxy-patch.json")"
        kubectl -n stackrox set env deploy/sensor -e ROX_CENTRAL_ENDPOINT_NO_PROXY="$original_endpoint" -e ROX_CENTRAL_ENDPOINT="localhost:8989"
    fi

    echo
}
