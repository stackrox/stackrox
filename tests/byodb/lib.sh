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
        numDeployments="$(curl -sSk --config <(curl_cfg user "admin:$ROX_ADMIN_PASSWORD") -X POST -d "{\"operationName\":\"summary_counts\",\"variables\":{},\"query\":\"query summary_counts {\n  clusterCount\n  nodeCount\n  violationCount\n  deploymentCount\n  imageCount\n  secretCount\n}\"}" "https://$API_ENDPOINT/api/graphql" | jq '.data.deploymentCount' -r)"
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
      if (( $(date '+%s') - start_time > 1200 )); then
        kubectl -n stackrox get pod -o wide
        kubectl -n stackrox get deploy -o wide
        echo >&2 "Timed out after 20m"
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

    rm -f FAIL
    remove_qa_test_results

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
  curl --config <(curl_cfg user "admin:${ROX_ADMIN_PASSWORD}") -k "https://${API_ENDPOINT}${url}" "$@"
}

deploy_external_postgres() {
    pwd
    EXTERNAL_DB_PASSWORD="$(tr -dc _A-Z-a-z-0-9 < /dev/urandom | head -c12 || true)"
    EXTERNAL_DB_USER=stackrox
    ci_export "EXTERNAL_DB_PASSWORD" "$EXTERNAL_DB_PASSWORD"
    ci_export "EXTERNAL_DB_USER" "$EXTERNAL_DB_USER"

    kubectl create namespace stackrox
    envsubst < ./tests/byodb/simple-postgres.yaml | kubectl apply -f -

    kubectl get pods -n stackrox
    sleep 60
}

deploy_external_postgres_central() {
#    make cli
    deploy_external_postgres

    PATH="bin/$TEST_HOST_PLATFORM:$PATH" command -v roxctl
    PATH="bin/$TEST_HOST_PLATFORM:$PATH" roxctl version

    # Let's try helm
    ROX_ADMIN_PASSWORD="$(tr -dc _A-Z-a-z-0-9 < /dev/urandom | head -c12 || true)"
    PATH="bin/$TEST_HOST_PLATFORM:$PATH" roxctl helm output central-services --image-defaults opensource --output-dir /tmp/early-stackrox-central-services-chart --remove

    helm install -n stackrox --create-namespace stackrox-central-services /tmp/early-stackrox-central-services-chart \
         --set central.adminPassword.value="${ROX_ADMIN_PASSWORD}" \
         --set central.db.password.value="${EXTERNAL_DB_PASSWORD}" \
         --set central.db.enabled=true \
         --set central.db.external=true \
         --set central.db.source.connectionString: "host=postgres client_encoding=UTF8 user=${EXTERNAL_DB_USER} dbname=stackrox statement_timeout=1200000" \
         --set central.persistence.none=true \
         --set central.exposure.loadBalancer.enabled=true \
         --set system.enablePodSecurityPolicies=false \
         --set central.image.tag="${CURRENT_TAG}" \
         --set scanner.image.tag="$(cat SCANNER_VERSION)" \
         --set scanner.dbImage.tag="$(cat SCANNER_VERSION)" \
         --set scanner.resources.limits.memory="6Gi"

    # Installing this way returns faster than the scripts but everything isn't running when it finishes like with
    # the scripts.  So we will give it a minute for things to get started before we proceed
    sleep 60

    ROX_USERNAME="admin"
    ci_export "ROX_USERNAME" "$ROX_USERNAME"
    ci_export "ROX_ADMIN_PASSWORD" "$ROX_ADMIN_PASSWORD"
}

restore_4_6_backup() {
    info "Restoring a 4.6 backup into a newer central"

    restore_4_6_postgres_backup
}

preamble() {
    info "Starting test preamble"

    if is_darwin; then
        HOST_OS="darwin"
    elif is_linux; then
        HOST_OS="linux"
    else
        die "Only linux or darwin are supported for this test"
    fi

    case "$(uname -m)" in
        x86_64) TEST_HOST_PLATFORM="${HOST_OS}_amd64" ;;
        aarch64) TEST_HOST_PLATFORM="${HOST_OS}_arm64" ;;
        arm64) TEST_HOST_PLATFORM="${HOST_OS}_arm64" ;;
        ppc64le) TEST_HOST_PLATFORM="${HOST_OS}_ppc64le" ;;
        s390x) TEST_HOST_PLATFORM="${HOST_OS}_s390x" ;;
        *) die "Unknown architecture" ;;
    esac

    require_executable "$TEST_ROOT/bin/${TEST_HOST_PLATFORM}/roxctl"

    if is_CI; then
        if ! command -v yq >/dev/null 2>&1; then
            sudo wget https://github.com/mikefarah/yq/releases/download/v4.4.1/yq_linux_amd64 -O /usr/bin/yq
            sudo chmod 0755 /usr/bin/yq
        fi
    else
        require_executable yq
    fi
}
