#!/usr/bin/env bash
# This file is only sourced, but the following line helps set the stage for shellcheck.
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

pushd "$DIR"

export DEFAULT_IMAGE_REGISTRY="${DEFAULT_IMAGE_REGISTRY:-"$(make --quiet --no-print-directory -C "$(git rev-parse --show-toplevel)" default-image-registry)"}"
echo "DEFAULT_IMAGE_REGISTRY set to $DEFAULT_IMAGE_REGISTRY"

export MAIN_IMAGE_REPO="${MAIN_IMAGE_REPO:-$DEFAULT_IMAGE_REGISTRY/main}"
echo "MAIN_IMAGE_REPO set to $MAIN_IMAGE_REPO"

export MAIN_IMAGE_TAG="${MAIN_IMAGE_TAG:-$(make --quiet --no-print-directory -C "$(git rev-parse --show-toplevel)" tag)}"
echo "StackRox image tag set to $MAIN_IMAGE_TAG"

export MAIN_IMAGE="${MAIN_IMAGE_REPO}:${MAIN_IMAGE_TAG}"
echo "StackRox image set to $MAIN_IMAGE"

export ROX_POSTGRES_DATASTORE="true"
echo "ROX_POSTGRES_DATASTORE set to $ROX_POSTGRES_DATASTORE"

export CENTRAL_DB_IMAGE_REPO="${CENTRAL_DB_IMAGE_REPO:-$DEFAULT_IMAGE_REGISTRY/central-db}"
echo "CENTRAL_DB_IMAGE_REPO set to $CENTRAL_DB_IMAGE_REPO"

export CENTRAL_DB_IMAGE_TAG="${CENTRAL_DB_IMAGE_TAG:-${MAIN_IMAGE_TAG}}"
echo "StackRox central db image tag set to $CENTRAL_DB_IMAGE_TAG"

export CENTRAL_DB_IMAGE="${CENTRAL_DB_IMAGE:-${CENTRAL_DB_IMAGE_REPO}:${CENTRAL_DB_IMAGE_TAG}}"
echo "StackRox central db image set to $CENTRAL_DB_IMAGE"

export ROXCTL_IMAGE_REPO="${ROXCTL_IMAGE_REPO:-$DEFAULT_IMAGE_REGISTRY/roxctl}"
echo "ROXCTL_IMAGE_REPO set to $ROXCTL_IMAGE_REPO"

export ROXCTL_IMAGE_TAG="${ROXCTL_IMAGE_TAG:-${MAIN_IMAGE_TAG}}"
echo "StackRox roxctl image tag set to $ROXCTL_IMAGE_TAG"

export ROXCTL_IMAGE="${ROXCTL_IMAGE_REPO}:${ROXCTL_IMAGE_TAG}"
echo "StackRox roxctl image set to $ROXCTL_IMAGE"

export ROXCTL_ROX_IMAGE_FLAVOR="${ROXCTL_ROX_IMAGE_FLAVOR:-$(make --quiet --no-print-directory -C "$(git rev-parse --show-toplevel)" image-flavor)}"
echo "Image flavor for roxctl set to $ROXCTL_ROX_IMAGE_FLAVOR"

popd

# curl_central_once
# Runs curl with --silent --show-error --insecure.
# Obeys $ROX_ADMIN_USER and $ROX_ADMIN_PASSWORD.
# arguments:
#   - anything else `curl` will accept, including the URL
function curl_central_once() {
    local cmd=(curl --silent --show-error --insecure)
    local admin_user="${ROX_ADMIN_USER:-admin}"
    if [[ -n "${ROX_ADMIN_PASSWORD:-}" ]]; then
        cmd+=(-u "${admin_user}:${ROX_ADMIN_PASSWORD}")
    fi
    "${cmd[@]}" "$@"
}


# curl_central_retry
# Runs curl like curl_central_once, but with retries and --fail.
# Captures stdout and emits it only once (for the last invocation of curl).
# arguments:
#   - anything else `curl` will accept, including the URL
function curl_central_retry() {
    local tmp_out
    tmp_out="$(mktemp)"
    local delay_sec=10
    # On top of --retry we do our own retry loop, to make sure no corner cases slip through.
    # https://github.com/curl/curl/issues/6712#issuecomment-796534491
    for _ in $(seq 3); do
        if curl_central_once --fail --retry 3 --retry-delay "${delay_sec}" --retry-connrefused "$@" > "${tmp_out}"; then
            cat "${tmp_out}"
            rm -f "${tmp_out}"
            return 0
        fi
        sleep "${delay_sec}"
    done
    cat "${tmp_out}"
    rm -f "${tmp_out}"
    return 1
}

# generate_ca
# arguments:
#   - directory to drop files in
function generate_ca {
    OUTPUT_DIR="$1"

    if [ ! -f "$OUTPUT_DIR/ca-key.pem" ]; then
        echo "Generating CA key..."
        echo " + Getting cfssl..."
        go install github.com/cloudflare/cfssl/cmd/...@latest
        echo " + Generating keypair..."
        PWD=$(pwd)
        cd "$OUTPUT_DIR"
        echo '{"CN":"CA","key":{"algo":"ecdsa"}}' | cfssl gencert -initca - | cfssljson -bare ca -
        cd "$PWD"
    fi
    echo
}

# wait_for_central
# arguments:
#   - API server endpoint to ping
function wait_for_central {
    LOCAL_API_ENDPOINT="$1"
    local central_namespace=${2:-stackrox}

    echo -n "Waiting for Central in namespace ${central_namespace} to respond."
    local start_time
    start_time="$(date '+%s')"
    local deadline=$((start_time + 10*60))  # 10 minutes
    until curl_central_once --output /dev/null --fail "https://$LOCAL_API_ENDPOINT/v1/ping"; do
        if [[ "$(date '+%s')" -gt "$deadline" ]]; then
            echo >&2 "Exceeded deadline waiting for Central, aborting its process."
            "${ORCH_CMD}" -n "${central_namespace}" exec deployment/central -c central -- kill -ABRT 1
            exit 1
        fi
        echo -n '.'
        sleep 1
    done
    echo
}

# get_cluster_zip
# arguments:
#   - central API server endpoint reachable from this host
#   - name of cluster
#   - type of cluster (e.g., KUBERNETES_CLUSTER)
#   - image reference (e.g., stackrox/main:$(make tag))
#   - central API endpoint reachable from the container (e.g., my-host:8080)
#   - directory to drop files in
#   - collection method
#   - extra fields in JSON format (a leading comma MUST be added; a trailing comma must NOT be added)
function get_cluster_zip {
    LOCAL_API_ENDPOINT="$1"
    CLUSTER_NAME="$2"
    CLUSTER_TYPE="$3"
    CLUSTER_IMAGE="$4"
    CLUSTER_API_ENDPOINT="$5"
    OUTPUT_DIR="$6"
    COLLECTION_METHOD="$7"
    EXTRA_JSON="$8"

    COLLECTION_METHOD_ENUM="default"
    if [[ "$COLLECTION_METHOD" == "core_bpf" ]]; then
       COLLECTION_METHOD_ENUM="CORE_BPF"
    elif [[ "$COLLECTION_METHOD" == "ebpf" ]]; then
      COLLECTION_METHOD_ENUM="EBPF"
    elif [[ "$COLLECTION_METHOD" == "none" ]]; then
      COLLECTION_METHOD_ENUM="NO_COLLECTION"
    else
      echo "Unknown collection method '$COLLECTION_METHOD', using default collection-method."
    fi

    echo "Creating a new cluster"
    export CLUSTER_JSON="{\"name\": \"$CLUSTER_NAME\", \"type\": \"$CLUSTER_TYPE\", \"main_image\": \"$CLUSTER_IMAGE\", \"central_api_endpoint\": \"$CLUSTER_API_ENDPOINT\", \"collection_method\": \"$COLLECTION_METHOD_ENUM\", \"admission_controller\": $ADMISSION_CONTROLLER $EXTRA_JSON}"

    TMP=$(mktemp)
    STATUS=$(curl_central_retry -X POST \
        -d "$CLUSTER_JSON" \
        -o "$TMP" \
        -w "%{http_code}\n" \
        "https://$LOCAL_API_ENDPOINT/v1/clusters")
    >&2 echo "Status: $STATUS"
    if [ "$STATUS" != "200" ]; then
      cat "$TMP"
      exit 1
    fi

    ID="$(jq -r .cluster.id "${TMP}")"

    echo "Getting zip file for cluster ${ID}"
    STATUS=$(curl_central_retry -X POST \
        -d "{\"id\": \"$ID\", \"createUpgraderSA\": true}" \
        -o "$OUTPUT_DIR/sensor-deploy.zip" \
        -w "%{http_code}\n" \
        "https://$LOCAL_API_ENDPOINT/api/extensions/clusters/zip")
    echo "Status: $STATUS"
    echo "Saved zip file to $OUTPUT_DIR"
    echo
}

function setup_internal_sso() {
    local LOCAL_API_ENDPOINT="$1"
    local LOCAL_CLIENT_SECRET="$2"
	echo "Setting up Dev Internal SSO login"

    roxctl declarative-config create auth-provider oidc \
        --secret=sensitive-declarative-configurations \
        --namespace=stackrox \
        --name="Internal-SSO" \
        --ui-endpoint="${LOCAL_API_ENDPOINT}" \
        --minimum-access-role=Admin \
        --extra-ui-endpoints=localhost:8000 \
        --extra-ui-endpoints=localhost:3000 \
        --extra-ui-endpoints=localhost:8443 \
        --issuer=https://auth.redhat.com/auth/realms/EmployeeIDP \
        --mode=post \
        --client-id=rhacs-dev-envs \
        --client-secret="${LOCAL_CLIENT_SECRET}" \
        --disable-offline-access=true
}
