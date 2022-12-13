#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Test utility functions for upgrades

wait_for_central_reconciliation() {
    info "Waiting for central reconciliation"

    # Reconciliation is rather slow in this case, since the central has a DB with a bunch of deployments,
    # none of which exist. So when sensor connects, the reconciliation deletion takes a while to flush.
    # This causes flakiness with the smoke tests.
    # To mitigate this, wait for the deployments to get deleted before running the tests.
    local success=0
    for i in $(seq 1 90); do
        local numDeployments
        numDeployments="$(curl -sSk -u "admin:$ROX_PASSWORD" "https://$API_ENDPOINT/v1/summary/counts" | jq '.numDeployments' -r)"
        echo "Try number ${i}. Number of deployments in Central: $numDeployments"
        [[ -n "$numDeployments" ]]
        if [[ "$numDeployments" -lt 100 ]]; then
            success=1
            break
        fi
        sleep 10
    done
    [[ "$success" == 1 ]]
}

wait_for_scanner_to_be_ready() {
    echo "Waiting for scanner to be ready"
    start_time="$(date '+%s')"
    while true; do
      scanner_json="$(kubectl -n stackrox get deploy/scanner -o json)"
      replicas="$(jq '.status.replicas' <<<"${scanner_json}")"
      readyReplicas="$(jq '.status.readyReplicas' <<<"${scanner_json}")"
      echo "scanner replicas: $replicas"
      echo "scanner readyReplicas: $readyReplicas"
      if [[  "$replicas" == "$readyReplicas" ]]; then
        break
      fi
      if (( $(date '+%s') - start_time > 300 )); then
        kubectl -n stackrox get pod -o wide
        kubectl -n stackrox get deploy -o wide
        echo >&2 "Timed out after 5m"
        exit 1
      fi
      sleep 5
    done
    echo "Scanner is ready"
}

validate_upgrade() {
    if [[ "$#" -ne 3 ]]; then
        die "missing args. usage: validate_upgrade <stage name> <stage description> <upgrade_cluster_id>"
    fi

    local stage_name="$1"
    local stage_description="$2"
    local upgrade_cluster_id="$3"
    local policies_dir="../pkg/defaults/policies/files"

    if [[ -n "${API_TOKEN:-}" ]]; then
        info "Verifying API token generated can access the central"
        echo "${API_TOKEN}" | "${TEST_ROOT}/bin/${TEST_HOST_PLATFORM}/roxctl" --insecure-skip-tls-verify --insecure -e "${API_ENDPOINT}" --token-file /dev/stdin central whoami > /dev/null
    fi

    info "Validating the upgrade with upgrade tests: $stage_description"

    CLUSTER="$CLUSTER_TYPE_FOR_TEST" \
        UPGRADE_CLUSTER_ID="$upgrade_cluster_id" \
        POLICIES_JSON_RELATIVE_PATH="$policies_dir" \
        make -C qa-tests-backend upgrade-test || touch FAIL
    store_qa_test_results "validate-upgrade-tests-${stage_name}"
    [[ ! -f FAIL ]] || die "Upgrade tests failed"
}

function roxcurl() {
  local url="$1"
  shift
  curl -u "admin:${ROX_PASSWORD}" -k "https://${API_ENDPOINT}${url}" "$@"
}

deploy_earlier_central() {
    info "Deploying: $EARLIER_TAG..."

    mkdir -p "bin/$TEST_HOST_PLATFORM"
    if is_CI; then
        gsutil cp "gs://stackrox-ci/roxctl-$EARLIER_TAG" "bin/$TEST_HOST_PLATFORM/roxctl"
    else
        make cli
    fi
    chmod +x "bin/$TEST_HOST_PLATFORM/roxctl"
    PATH="bin/$TEST_HOST_PLATFORM:$PATH" command -v roxctl
    PATH="bin/$TEST_HOST_PLATFORM:$PATH" roxctl version
    PATH="bin/$TEST_HOST_PLATFORM:$PATH" \
    MAIN_IMAGE_TAG="$EARLIER_TAG" \
    SCANNER_IMAGE="$REGISTRY/scanner:$(cat SCANNER_VERSION)" \
    SCANNER_DB_IMAGE="$REGISTRY/scanner-db:$(cat SCANNER_VERSION)" \
    ./deploy/k8s/central.sh

    get_central_basic_auth_creds
}

restore_backup_test() {
    info "Restoring a 56.1 backup into a newer central"

    restore_56_1_backup
}

force_rollback() {
    info "Forcing a rollback to $FORCE_ROLLBACK_VERSION"

    local upgradeStatus
    upgradeStatus=$(curl -sSk -X GET -u "admin:${ROX_PASSWORD}" https://"${API_ENDPOINT}"/v1/centralhealth/upgradestatus)
    echo "upgrade status: ${upgradeStatus}"
    test_equals_non_silent "$(echo "$upgradeStatus" | jq '.upgradeStatus.version' -r)" "$(make --quiet tag)"
    test_equals_non_silent "$(echo "$upgradeStatus" | jq '.upgradeStatus.forceRollbackTo' -r)" "$FORCE_ROLLBACK_VERSION"
    test_equals_non_silent "$(echo "$upgradeStatus" | jq '.upgradeStatus.canRollbackAfterUpgrade' -r)" "true"
    test_gt_non_silent "$(echo "$upgradeStatus" | jq '.upgradeStatus.spaceAvailableForRollbackAfterUpgrade' -r)" "$(echo "$upgradeStatus" | jq '.upgradeStatus.spaceRequiredForRollbackAfterUpgrade' -r)"

    kubectl -n stackrox get configmap/central-config -o yaml | yq e '{"data": .data}' - >/tmp/force_rollback_patch
    local central_config
    central_config=$(yq e '.data["central-config.yaml"]' /tmp/force_rollback_patch | yq e ".maintenance.forceRollbackVersion = \"$FORCE_ROLLBACK_VERSION\"" -)
    local config_patch
    config_patch=$(yq e ".data[\"central-config.yaml\"] |= \"$central_config\"" /tmp/force_rollback_patch)
    echo "config patch: $config_patch"

    kubectl -n stackrox patch configmap/central-config -p "$config_patch"
    kubectl -n stackrox set image deploy/central "central=$REGISTRY/main:$FORCE_ROLLBACK_VERSION"
}

validate_sensor_bundle_via_upgrader() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: validate_sensor_bundle_via_upgrader <deploy_dir>"
    fi

    local deploy_dir="$1"

    info "Validating the sensor bundle via upgrader"

    kubectl proxy --port 28001 &
    local proxy_pid=$!
    sleep 5

    KUBECONFIG="$TEST_ROOT/scripts/ci/kube-api-proxy/config.yml" \
        "$TEST_ROOT/bin/${TEST_HOST_PLATFORM}/upgrader" \
        -kube-config kubectl \
        -local-bundle "$deploy_dir/sensor-deploy" \
        -workflow validate-bundle || {
            kill "$proxy_pid" || true
            save_junit_failure "Validate_Sensor_Bundle_Via_Upgrader" \
                "Failed" \
                "Check build_log"
            return 1
        }

    kill "$proxy_pid"
}

test_sensor_bundle() {
    info "Testing the sensor bundle"

    rm -rf sensor-remote
    "$TEST_ROOT/bin/${TEST_HOST_PLATFORM}/roxctl" -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor get-bundle remote
    [[ -d sensor-remote ]]

    ./sensor-remote/sensor.sh

    kubectl -n stackrox patch deploy/sensor --patch '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"limits":{"cpu":"500m","memory":"500Mi"},"requests":{"cpu":"500m","memory":"500Mi"}}}]}}}}'

    sensor_wait

    ./sensor-remote/delete-sensor.sh
    rm -rf sensor-remote
}

test_upgrader() {
    info "Starting bin/upgrader tests"

    deactivate_metrics_server

    info "Creating a 'sensor-remote-new' cluster"

    rm -rf sensor-remote-new
    "$TEST_ROOT/bin/${TEST_HOST_PLATFORM}/roxctl" -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor generate k8s \
        --main-image-repository "${MAIN_IMAGE_REPO:-$REGISTRY/main}" \
        --collector-image-repository "${COLLECTOR_IMAGE_REPO:-$REGISTRY/collector}" \
        --name remote-new \
        --create-admission-controller

    deploy_sensor_via_upgrader "for the first time, to test rollback" 3b2cbf78-d35a-4c2c-b67b-e37f805c14da
    rollback_sensor_via_upgrader 3b2cbf78-d35a-4c2c-b67b-e37f805c14da

    deploy_sensor_via_upgrader "from scratch" 9b8f2cbb-72c6-4e00-b375-5b50ce7f988b

    deploy_sensor_via_upgrader "again, but with the same upgrade process ID" 9b8f2cbb-72c6-4e00-b375-5b50ce7f988b

    deploy_sensor_via_upgrader "yet again, but with a new upgrade process ID" 789c9262-5dd3-4d58-a824-c2a099892bd6

    webhook_timeout_before_patch="$(kubectl -n stackrox get validatingwebhookconfiguration/stackrox -o json | jq '.webhooks | .[0] | .timeoutSeconds')"
    echo "Webhook timeout before patch: ${webhook_timeout_before_patch}"
    webhook_timeout_after_patch="$((webhook_timeout_before_patch + 1))"
    echo "Desired webhook timeout after patch: ${webhook_timeout_after_patch}"

    info "Patch admission webhook"
    kubectl -n stackrox patch validatingwebhookconfiguration stackrox --type 'json' -p "[{'op':'replace','path':'/webhooks/0/timeoutSeconds','value':${webhook_timeout_after_patch}}]"
    if [[ "$(kubectl -n stackrox get validatingwebhookconfiguration/stackrox -o json | jq '.webhooks | .[0] | .timeoutSeconds')" -ne "${webhook_timeout_after_patch}" ]]; then
        echo "Webhook not patched"
        kubectl -n stackrox get validatingwebhookconfiguration/stackrox -o yaml
        exit 1
    fi

    info "Patch resources"
    kubectl -n stackrox set resources deploy/sensor -c sensor --requests 'cpu=1.1,memory=1.1Gi'

    deploy_sensor_via_upgrader "after manually patching webhook" 060a9fa6-0ed6-49ac-b70c-9ca692614707

    info "Verify the webhook was patched back by the upgrader"
    if [[ "$(kubectl -n stackrox get validatingwebhookconfiguration/stackrox -o json | jq '.webhooks | .[0] | .timeoutSeconds')" -ne "${webhook_timeout_before_patch}" ]]; then
        echo "Webhook not patched"
        kubectl -n stackrox get validatingwebhookconfiguration/stackrox -o yaml
        exit 1
    fi

    deploy_sensor_via_upgrader "with yet another new ID" 789c9262-5dd3-4d58-a824-c2a099892bd7

    info "Verify resources were patched back by the upgrader"
    resources="$(kubectl -n stackrox get deploy/sensor -o 'jsonpath=cpu={.spec.template.spec.containers[?(@.name=="sensor")].resources.requests.cpu},memory={.spec.template.spec.containers[?(@.name=="sensor")].resources.requests.memory}')"
    if [[ "$resources" != 'cpu=1,memory=1Gi' ]]; then
        echo "Resources ($resources) not patched back!"
        kubectl -n stackrox get deploy/sensor -o yaml
        exit 1
    fi

    info "Patch resources and add preserve resources annotation. Also, check toleration preservation"
    kubectl -n stackrox annotate deploy/sensor "auto-upgrade.stackrox.io/preserve-resources=true"
    kubectl -n stackrox set resources deploy/sensor -c sensor --requests 'cpu=1.1,memory=1.1Gi'
    kubectl -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"tolerations":[{"effect":"NoSchedule","key":"thekey","operator":"Equal","value":"thevalue"}]}}}}'

    deploy_sensor_via_upgrader "after patching resources with preserve annotation" 789c9262-5dd3-4d58-a824-c2a099892bd8

    info "Verify resources were not patched back by the upgrader"
    resources="$(kubectl -n stackrox get deploy/sensor -o 'jsonpath=cpu={.spec.template.spec.containers[?(@.name=="sensor")].resources.requests.cpu},memory={.spec.template.spec.containers[?(@.name=="sensor")].resources.requests.memory}')"
    if [[ "$resources" != 'cpu=1100m,memory=1181116006400m' ]]; then
        echo "Resources ($resources) appear patched back!"
        kubectl -n stackrox get deploy/sensor -o yaml
        exit 1
    fi
    toleration="$(kubectl -n stackrox get deploy/sensor -o json | jq -rc '.spec.template.spec.tolerations[0] | (.effect + "," + .key + "," + .operator + "," + .value)')"
    echo "Found toleration: $toleration"
    if [[ "$toleration" != 'NoSchedule,thekey,Equal,thevalue' ]]; then
        echo "Tolerations were not passed through to new Sensor"
        kubectl -n stackrox get deploy/sensor -o yaml
        exit 1
    fi

    # It's important to re-activate the metrics server because it might place a finalizer on namespaces. If it isn't
    # active, namespaces might get stuck in the Terminating state.
    activate_metrics_server
}

deactivate_metrics_server() {
    info "Deactivate the metrics server by scaling it to 0 in order to reproduce ROX-4429"

    # This should already be in the API resources
    echo "Waiting for metrics.k8s.io to be in kubectl API resources..."
    local success=0
    for i in $(seq 1 10); do
        if kubectl api-resources 2>&1 | sed -e 's/^/out: /' | grep metrics.k8s.io; then
            success=1
            break
        fi
        sleep 5
    done
    [[ "$success" -eq 1 ]]

    kubectl -n kube-system scale deploy -l k8s-app=metrics-server --replicas=0

    echo "Waiting for metrics.k8s.io to NOT be in kubectl API resources..."
    local success=0
    # shellcheck disable=SC2034
    for i in $(seq 1 10); do
        kubectl api-resources >stdout.out 2>stderr.out || true
        if grep -q 'metrics.k8s.io.*the server is currently unable to handle the request' stderr.out; then
            success=1
            break
        fi
        echo "metrics.k8s.io still in API resources. Will try again..."
        cat stdout.out
        sed -e 's/^/out: /' < stderr.out # (prefix output to avoid triggering prow log focus)
        sleep 5
    done
    [[ "$success" -eq 1 ]]
    rm -f stdout.out stderr.out

    info "deactivated"
}

activate_metrics_server() {
    info "Activating the previously deactivated metrics server"

    # Ideally we would restore the previous replica count, but 1 works just fine
    kubectl -n kube-system scale deploy -l k8s-app=metrics-server --replicas=1

    echo "Waiting for metrics.k8s.io to be in kubectl API resources..."
    local success=0
    # shellcheck disable=SC2034
    for i in $(seq 1 30); do
        if kubectl api-resources 2>&1 | sed -e 's/^/out: /' | grep metrics.k8s.io; then
            success=1
            break
        fi
        sleep 5
    done
    [[ "$success" -eq 1 ]]

    info "activated"
}

deploy_sensor_via_upgrader() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: deploy_sensor_via_upgrader <stage> <upgrade_process_id>"
    fi

    local stage="$1"
    local upgrade_process_id="$2"

    info "Deploying sensor via upgrader: $stage"

    kubectl proxy --port 28001 &
    local proxy_pid=$!
    sleep 5

    ROX_UPGRADE_PROCESS_ID="$upgrade_process_id" \
        ROX_CENTRAL_ENDPOINT="$API_ENDPOINT" \
        ROX_MTLS_CA_FILE="$TEST_ROOT/sensor-remote-new/ca.pem" \
        ROX_MTLS_CERT_FILE="$TEST_ROOT/sensor-remote-new/sensor-cert.pem" \
        ROX_MTLS_KEY_FILE="$TEST_ROOT/sensor-remote-new/sensor-key.pem" \
        KUBECONFIG="$TEST_ROOT/scripts/ci/kube-api-proxy/config.yml" \
        "$TEST_ROOT/bin/${TEST_HOST_PLATFORM}/upgrader" -workflow roll-forward -local-bundle sensor-remote-new -kube-config kubectl || {
            kill "$proxy_pid" || true
            save_junit_failure "Deploy_Sensor_Via_Upgrader" \
                "Failed: $stage" \
                "Check build_log"
            return 1
        }

    kill "$proxy_pid"

    sensor_wait
}

rollback_sensor_via_upgrader() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: rollback_sensor_via_upgrader <upgrade_process_id>"
    fi

    local upgrade_process_id="$1"

    info "Rolling back sensor via upgrader"

    kubectl proxy --port 28001 &
    local proxy_pid=$!
    sleep 5

    ROX_UPGRADE_PROCESS_ID="$upgrade_process_id" \
        ROX_CENTRAL_ENDPOINT="$API_ENDPOINT" \
        ROX_MTLS_CA_FILE="$TEST_ROOT/sensor-remote-new/ca.pem" \
        ROX_MTLS_CERT_FILE="$TEST_ROOT/sensor-remote-new/sensor-cert.pem" \
        ROX_MTLS_KEY_FILE="$TEST_ROOT/sensor-remote-new/sensor-key.pem" \
        KUBECONFIG="$TEST_ROOT/scripts/ci/kube-api-proxy/config.yml" \
        "$TEST_ROOT/bin/${TEST_HOST_PLATFORM}/upgrader" -workflow roll-back -kube-config kubectl || {
            kill "$proxy_pid" || true
            save_junit_failure "Rollback_Sensor_Via_Upgrader" \
                "Failed" \
                "Check build_log"
            return 1
        }

    kill "$proxy_pid"
}

