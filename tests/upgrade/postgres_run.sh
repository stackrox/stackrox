#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Tests upgrade to Postgres.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

source "$TEST_ROOT/scripts/lib.sh"
source "$TEST_ROOT/scripts/ci/lib.sh"
source "$TEST_ROOT/scripts/ci/sensor-wait.sh"
source "$TEST_ROOT/tests/scripts/setup-certs.sh"
source "$TEST_ROOT/tests/e2e/lib.sh"
source "$TEST_ROOT/tests/upgrade/lib.sh"

test_upgrade() {
    info "Starting Postgres upgrade test"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade <log-output-dir>"
    fi

    local log_output_dir="$1"

    require_environment "KUBECONFIG"

    export_test_environment

    REPO_FOR_TIME_TRAVEL="/tmp/rox-postgres-upgrade-test"
    DEPLOY_DIR="deploy/k8s"
    QUAY_REPO="rhacs-eng"
    REGISTRY="quay.io/$QUAY_REPO"

    export OUTPUT_FORMAT="helm"
    export STORAGE="pvc"
    export CLUSTER_TYPE_FOR_TEST=K8S

    export ROXCTL_IMAGE_REPO="quay.io/$QUAY_REPO/roxctl"
    if is_CI; then
        require_environment "LONGTERM_LICENSE"
        export ROX_LICENSE_KEY="${LONGTERM_LICENSE}"
    fi

    preamble
    setup_deployment_env false false
    remove_existing_stackrox_resources
    test_upgrade_paths "$log_output_dir"
}

preamble() {
    info "Starting test preamble"

    if is_darwin; then
        TEST_HOST_OS="darwin"
    elif is_linux; then
        TEST_HOST_OS="linux"
    else
        die "Only linux or darwin are supported for this test"
    fi

    require_executable "$TEST_ROOT/bin/$TEST_HOST_OS/roxctl"

    info "Will clone or update a clean copy of the rox repo for test at $REPO_FOR_TIME_TRAVEL"
    if [[ -d "$REPO_FOR_TIME_TRAVEL" ]]; then
        if is_CI; then
          die "Repo for time travel already exists! This is unexpected in CI."
        fi
        (cd "$REPO_FOR_TIME_TRAVEL" && git checkout master && git reset --hard && git pull)
    else
        (cd "$(dirname "$REPO_FOR_TIME_TRAVEL")" && git clone https://github.com/stackrox/stackrox.git "$(basename "$REPO_FOR_TIME_TRAVEL")")
    fi

    if is_CI; then
        if ! command -v yq >/dev/null 2>&1; then
            sudo wget https://github.com/mikefarah/yq/releases/download/v4.4.1/yq_linux_amd64 -O /usr/bin/yq
            sudo chmod 0755 /usr/bin/yq
        fi
    else
        require_executable yq
    fi
}

test_upgrade_paths() {
    info "Testing various upgrade paths"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade_paths <log-output-dir>"
    fi

    local log_output_dir="$1"

    EARLIER_SHA="9f82d2713cfec4b5c876d8dc0149f6d9cd70d349"
    EARLIER_TAG="3.63.x-163-g2c4fe1563c"
    FORCE_ROLLBACK_VERSION="$EARLIER_TAG"

    cd "$REPO_FOR_TIME_TRAVEL"
    git checkout "$EARLIER_SHA"

    deploy_earlier_central
    wait_for_api

    export API_TOKEN="$(roxcurl /v1/apitokens/generate -d '{"name": "helm-upgrade-test", "role": "Admin"}' | jq -r '.token')"

    cd "$TEST_ROOT"
    helm_upgrade_to_current_with_postgres
    wait_for_api

    info "Waiting for scanner to be ready"
    wait_for_scanner_to_be_ready

    validate_upgrade "00_upgrade" "central upgrade to postgres" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    collect_and_check_stackrox_logs "$log_output_dir" "00_initial_check"

    info "Bouncing central"
    kubectl -n stackrox delete po "$(kubectl -n stackrox get po -l app=central -o=jsonpath='{.items[0].metadata.name}')" --grace-period=0
    wait_for_api

    validate_upgrade "01-bounce-after-upgrade" "bounce after postgres upgrade" "268c98c6-e983-4f4e-95d2-9793cebddfd7"
    collect_and_check_stackrox_logs "$log_output_dir" "01_post_bounce"

    info "Fetching a sensor bundle for cluster 'remote'"
    rm -rf sensor-remote
    "$TEST_ROOT/bin/$TEST_HOST_OS/roxctl" -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor get-bundle remote
    [[ -d sensor-remote ]]

    info "Installing sensor"
    ./sensor-remote/sensor.sh
    kubectl -n stackrox set image deploy/sensor "*=$REGISTRY/main:$(make --quiet tag)"
    kubectl -n stackrox set image deploy/admission-control "*=$REGISTRY/main:$(make --quiet tag)"
    kubectl -n stackrox set image ds/collector "collector=$REGISTRY/collector:$(cat COLLECTOR_VERSION)" \
        "compliance=$REGISTRY/main:$(make --quiet tag)"

    sensor_wait

    wait_for_central_reconciliation

    info "Running smoke tests"
    CLUSTER="$CLUSTER_TYPE_FOR_TEST" make -C qa-tests-backend smoke-test || touch FAIL
    store_qa_test_results "upgrade-paths-smoke-tests"
    [[ ! -f FAIL ]] || die "Smoke tests failed"

    collect_and_check_stackrox_logs "$log_output_dir" "03_final"
}

# Need to move some of the functions to a lib if no change of course
deploy_earlier_central() {
    info "Deploying: $EARLIER_TAG..."

    mkdir -p "bin/$TEST_HOST_OS"
    if is_CI; then
        gsutil cp "gs://stackrox-ci/roxctl-$EARLIER_TAG" "bin/$TEST_HOST_OS/roxctl"
    else
        make cli
    fi
    chmod +x "bin/$TEST_HOST_OS/roxctl"
    PATH="bin/$TEST_HOST_OS:$PATH" command -v roxctl
    PATH="bin/$TEST_HOST_OS:$PATH" roxctl version
    PATH="bin/$TEST_HOST_OS:$PATH" \
    MAIN_IMAGE_TAG="$EARLIER_TAG" \
    SCANNER_IMAGE="$REGISTRY/scanner:$(cat SCANNER_VERSION)" \
    SCANNER_DB_IMAGE="$REGISTRY/scanner-db:$(cat SCANNER_VERSION)" \
    ./deploy/k8s/central.sh

    get_central_basic_auth_creds
}

helm_upgrade_to_current_with_postgres() {
    info "Helm upgrade to current"

    # use postgres
    export ROX_POSTGRES_DATASTORE="true"

    # Get opensource charts and convert to development_build to support release builds
    if is_CI; then
        roxctl helm output central-services --image-defaults opensource --output-dir /tmp/stackrox-central-services-chart
        sed -i 's#quay.io/stackrox-io#quay.io/rhacs-eng#' /tmp/stackrox-central-services-chart/internal/defaults.yaml
    else
        make cli
        roxctl helm output central-services --image-defaults opensource --output-dir /tmp/stackrox-central-services-chart --remove
        sed -i "" 's#quay.io/stackrox-io#quay.io/rhacs-eng#' /tmp/stackrox-central-services-chart/internal/defaults.yaml
    fi

    # enable postgres
    password=`echo ${RANDOM}_$(date +%s-%d-%M) |base64|cut -c 1-20`
    kubectl -n stackrox create secret generic central-db-password --from-literal=password=$password
    kubectl -n stackrox apply -f $TEST_ROOT/tests/upgrade/pvc.yaml
    create_db_tls_secret
    kubectl -n stackrox set env deploy/central ROX_POSTGRES_DATASTORE=true

    helm upgrade -n stackrox stackrox-central-services /tmp/stackrox-central-services-chart --set central.db.enabled=true

    kubectl -n stackrox get deploy -o wide
}

create_db_tls_secret() {
    echo "Create certificates for central db"

    cert_dir="$(mktemp -d)"
    # get root ca
    kubectl -n stackrox exec -it deployment/central -- cat /run/secrets/stackrox.io/certs/ca.pem > $cert_dir/ca.pem
    kubectl -n stackrox exec -it deployment/central -- cat /run/secrets/stackrox.io/certs/ca-key.pem > $cert_dir/ca.key
    # generate central-db certs
    openssl genrsa -out $cert_dir/key.pem 4096
    openssl req -new -key $cert_dir/key.pem -subj "/CN=CENTRAL_DB_SERVICE: Central DB" > $cert_dir/newreq
    echo subjectAltName = DNS:central-db.stackrox.svc > $cert_dir/extfile.cnf
    openssl x509 -sha256 -req -CA $cert_dir/ca.pem -CAkey $cert_dir/ca.key -CAcreateserial -out $cert_dir/cert.pem -in $cert_dir/newreq -extfile $cert_dir/extfile.cnf
    # create secret
    kubectl -n stackrox create secret generic central-db-tls --save-config --dry-run=client --from-file=$cert_dir/ca.pem --from-file=$cert_dir/cert.pem --from-file=$cert_dir/key.pem -o yaml | kubectl apply -f -
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_upgrade "$*"
fi
