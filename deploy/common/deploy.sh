#!/usr/bin/env bash

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

function curl_central() {
	cmd=(curl -k)
	local admin_user="${ROX_ADMIN_USER:-admin}"
	if [[ -n "${ROX_ADMIN_PASSWORD:-}" ]]; then
		cmd+=(-u "${admin_user}:${ROX_ADMIN_PASSWORD}")
	fi
	"${cmd[@]}" "$@"
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

    echo -n "Waiting for Central to respond."
    set +e
    local start_time
    start_time="$(date '+%s')"
    local deadline=$((start_time + 10*60))  # 10 minutes
    until curl_central --output /dev/null --silent --fail "https://$LOCAL_API_ENDPOINT/v1/ping"; do
        if [[ "$(date '+%s')" -gt "$deadline" ]]; then
            echo >&2 "Exceeded deadline waiting for Central."
            central_pod="$("${ORCH_CMD}" -n stackrox get pods -l app=central -ojsonpath='{.items[0].metadata.name}')"
            if [[ -n "$central_pod" ]]; then
                "${ORCH_CMD}" -n stackrox exec "${central_pod}" -c central -- kill -ABRT 1
            fi
            exit 1
        fi
        echo -n '.'
        sleep 1
    done
    set -e
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
    STATUS=$(curl_central -X POST \
        -d "$CLUSTER_JSON" \
        -s \
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
    STATUS=$(curl_central -X POST \
        -d "{\"id\": \"$ID\", \"createUpgraderSA\": true}" \
        -s \
        -o "$OUTPUT_DIR/sensor-deploy.zip" \
        -w "%{http_code}\n" \
        "https://$LOCAL_API_ENDPOINT/api/extensions/clusters/zip")
    echo "Status: $STATUS"
    echo "Saved zip file to $OUTPUT_DIR"
    echo
}

# get_identity
# arguments:
#   - central API server endpoint reachable from this host
#   - ID of a cluster that has already been created
#   - directory to drop files in
function get_identity {
    LOCAL_API_ENDPOINT="$1"
    CLUSTER_ID="$2"
    OUTPUT_DIR="$3"

    echo "Getting identity for new cluster"
    export ID_JSON="{\"id\": \"$CLUSTER_ID\", \"type\": \"SENSOR_SERVICE\"}"
    TMP=$(mktemp)
    STATUS=$(curl_central -X POST \
        -d "$ID_JSON" \
        -s \
        -o "$TMP" \
        -w "%{http_code}\n" \
        "https://$LOCAL_API_ENDPOINT/v1/serviceIdentities")
    echo "Status: $STATUS"
    echo "Response: $(cat "${TMP}")"
    jq -r .certificate "$TMP" > "$OUTPUT_DIR/sensor-cert.pem"
    jq -r .privateKey "$TMP" > "$OUTPUT_DIR/sensor-key.pem"
    rm "$TMP"
    echo
}

# get_authority
# arguments:
#   - central API server endpoint reachable from this host
#   - directory to drop files in
function get_authority {
    LOCAL_API_ENDPOINT="$1"
    OUTPUT_DIR="$2"

    echo "Getting CA certificate"
    TMP="$(mktemp)"
    STATUS=$(curl_central \
        -s \
        -o "$TMP" \
        -w "%{http_code}\n" \
        "https://$LOCAL_API_ENDPOINT/v1/authorities")
    echo "Status: $STATUS"
    echo "Response: $(cat "${TMP}")"
    jq -r .authorities[0].certificate "$TMP" > "$OUTPUT_DIR/ca.pem"
    rm "$TMP"
    echo
}

# get_central_feature_flag_enabled
# arguments:
#   - central API server endpoint reachable from this host
#   - envVar of feature flag to get
function get_central_feature_flag_enabled {
    local local_api_endpoint="$1"
    local env_var="$2"

    curl_central -s "https://${local_api_endpoint}/v1/featureflags" | \
        jq ".featureFlags[] | select(.envVar==\"${env_var}\") | .enabled"
}

function setup_license() {
    local local_api_endpoint="$1"
    local license_file="$2"

    local status
    status="$(curl_central \
	    -s \
	    -o /dev/null \
        "https://${local_api_endpoint}/v1/licenses/list" \
        -w "%{http_code}\n" || true)"
    if [[ "$status" == "404" ]]; then
        echo "Received a 404 response when querying license API. It looks like license enforcement is not enabled."
        return 0
    fi

    echo "Injecting license ..."
    [[ -f "$license_file" ]] || { echo "License file $license_file not found!" ; return 1 ; }

    local tmp
    tmp="$(mktemp)"
    status=$(curl_central \
	    -s \
	    -o "$tmp" \
        "https://${local_api_endpoint}/v1/licenses/add" \
        -w "%{http_code}\n" \
        -X POST \
        -d @- < <(
            jq -r -n '{"licenseKey": $key}' --arg key "$(cat "$license_file")"
        ) )
    local exit_code=$?
    echo "Status: $status"
    [[ "$exit_code" -eq 0 ]] || { echo "Failed to inject license" ; cat "$(tmp)" ; return 1 ; }
    echo "Waiting for central to restart ..."
    sleep 5
    wait_for_central "$local_api_endpoint"
}

function setup_auth0() {
    local LOCAL_API_ENDPOINT="$1"
    local LOCAL_CLIENT_SECRET="$2"
	echo "Setting up Dev Auth0 login"

	TMP=$(mktemp)
	STATUS=$(curl_central \
	    -s \
        -o "$TMP" \
        "https://${LOCAL_API_ENDPOINT}/v1/authProviders" \
        -w "%{http_code}\n" \
        -X POST \
        -d @- <<-EOF
{
	"name": "StackRox Dev (Auth0)",
	"type": "oidc",
	"uiEndpoint": "${LOCAL_API_ENDPOINT}",
	"enabled": true,
	"validated": true,
	"config": {
		"issuer": "https://sr-dev.auth0.com",
		"client_id": "bu63HaVAuVPEgMUeRVfL5PzrqTXaedA2",
		"client_secret": "${LOCAL_CLIENT_SECRET}",
		"mode": "post"
	},
	"extraUiEndpoints": ["localhost:8000", "localhost:3000", "localhost:8001"]
}
EOF
    )
    echo "Status: $STATUS"
    AUTH_PROVIDER_ID="$(jq <"$TMP" -r '.id')"
    echo "Created auth provider: ${AUTH_PROVIDER_ID}"

    echo "Setting up role for Auth0"
    curl_central -s "https://${LOCAL_API_ENDPOINT}/v1/groups" -X POST -d @- >/dev/null <<-EOF
{
    "props": {
        "authProviderId": "${AUTH_PROVIDER_ID}"
    },
    "roleName": "Admin"
}
EOF
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
