#!/usr/bin/env bash

export MAIN_IMAGE_TAG="${MAIN_IMAGE_TAG:-$(git describe --tags --abbrev=10 --dirty)}"
echo "StackRox image tag set to $MAIN_IMAGE_TAG"

export MAIN_IMAGE="${MAIN_IMAGE:-stackrox/main:$MAIN_IMAGE_TAG}"
echo "StackRox image set to $MAIN_IMAGE"

export SCANNER_IMAGE="${SCANNER_IMAGE:-stackrox/scanner:0.5.3}"
echo "StackRox scanner image set to $SCANNER_IMAGE"

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
        go get -u github.com/cloudflare/cfssl/cmd/...
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
    local start_time="$(date '+%s')"
    local deadline=$((start_time + 10*60))  # 10 minutes
    until $(curl_central --output /dev/null --silent --fail "https://$LOCAL_API_ENDPOINT/v1/ping"); do
        if [[ "$(date '+%s')" > "$deadline" ]]; then
            echo >&2 "Exceeded deadline waiting for Central."
            central_pod="$("${ORCH_CMD}" -n stackrox get pods -l app=central -ojsonpath={.items[0].metadata.name})"
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
#   - image reference (e.g., stackrox/main:$(git describe --tags --abbrev=10 --dirty))
#   - central API endpoint reachable from the container (e.g., my-host:8080)
#   - directory to drop files in
#   - extra fields in JSON format (a leading comma MUST be added; a trailing comma must NOT be added)
function get_cluster_zip {
    LOCAL_API_ENDPOINT="$1"
    CLUSTER_NAME="$2"
    CLUSTER_TYPE="$3"
    CLUSTER_IMAGE="$4"
    CLUSTER_API_ENDPOINT="$5"
    OUTPUT_DIR="$6"
    RUNTIME_SUPPORT="$7"
    EXTRA_JSON="$8"

    echo "Creating a new cluster"
    export CLUSTER_JSON="{\"name\": \"$CLUSTER_NAME\", \"type\": \"$CLUSTER_TYPE\", \"main_image\": \"$CLUSTER_IMAGE\", \"central_api_endpoint\": \"$CLUSTER_API_ENDPOINT\", \"runtime_support\": $RUNTIME_SUPPORT, \"admission_controller\": $ADMISSION_CONTROLLER $EXTRA_JSON}"

    TMP=$(mktemp)
    STATUS=$(curl_central -X POST \
        -d "$CLUSTER_JSON" \
        -s \
        -o $TMP \
        -w "%{http_code}\n" \
        https://$LOCAL_API_ENDPOINT/v1/clusters)
    >&2 echo "Status: $STATUS"
    if [ "$STATUS" != "200" ]; then
      cat $TMP
      exit 1
    fi

    ID="$(cat ${TMP} | jq -r .cluster.id)"

    echo "Getting zip file for cluster ${ID}"
    STATUS=$(curl_central -X POST \
        -d "{\"id\": \"$ID\"}" \
        -s \
        -o $OUTPUT_DIR/sensor-deploy.zip \
        -w "%{http_code}\n" \
        https://$LOCAL_API_ENDPOINT/api/extensions/clusters/zip)
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
        https://$LOCAL_API_ENDPOINT/v1/serviceIdentities)
    echo "Status: $STATUS"
    echo "Response: $(cat ${TMP})"
    cat "$TMP" | jq -r .certificate > "$OUTPUT_DIR/sensor-cert.pem"
    cat "$TMP" | jq -r .privateKey > "$OUTPUT_DIR/sensor-key.pem"
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
        https://$LOCAL_API_ENDPOINT/v1/authorities)
    echo "Status: $STATUS"
    echo "Response: $(cat ${TMP})"
    cat "$TMP" | jq -r .authorities[0].certificate > "$OUTPUT_DIR/ca.pem"
    rm "$TMP"
    echo
}

function setup_auth0() {
    local LOCAL_API_ENDPOINT="$1"
	echo "Setting up StackRox Dev Auth0 login"
	TMP=$(mktemp)
	STATUS=$(curl_central \
	    -s \
        -o $TMP \
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
		"mode": "post"
	},
	"extraUiEndpoints": ["localhost:3000", "prevent.stackrox.com"]
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
