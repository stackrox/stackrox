#!/usr/bin/env bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

export LOCAL_API_ENDPOINT="${LOCAL_API_ENDPOINT:-localhost:8080}"
echo "Local Apollo endpoint set to $LOCAL_API_ENDPOINT"

export APOLLO_IMAGE_TAG="${APOLLO_IMAGE_TAG:-latest}"
echo "Apollo image tag set to $APOLLO_IMAGE_TAG"

export APOLLO_IMAGE="${APOLLO_IMAGE:-stackrox/apollo:$APOLLO_IMAGE_TAG}"
echo "Apollo image set to $APOLLO_IMAGE"

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
    until $(curl --output /dev/null --silent --fail -k "https://$LOCAL_API_ENDPOINT/v1/ping"); do
        echo -n '.'
        sleep 1
    done
    set -e
    echo
}

# create_cluster
# arguments:
#   - central API server endpoint reachable from this host
#   - name of cluster
#   - type of cluster (e.g., SWARM_CLUSTER)
#   - image reference (e.g., stackrox/apollo:latest)
#   - central API endpoint reachable from the container (e.g., my-host:8080)
#   - directory to drop files in
#   - extra fields in JSON format
function create_cluster {
    LOCAL_API_ENDPOINT="$1"
    CLUSTER_NAME="$2"
    CLUSTER_TYPE="$3"
    CLUSTER_IMAGE="$4"
    CLUSTER_API_ENDPOINT="$5"
    OUTPUT_DIR="$6"
    EXTRA_JSON="$7"

    echo "Creating a new cluster"
    if [ "$EXTRA_JSON" != "" ]; then
        EXTRA_JSON=", $EXTRA_JSON"
    fi
    export CLUSTER_JSON="{\"name\": \"$CLUSTER_NAME\", \"type\": \"$CLUSTER_TYPE\", \"apollo_image\": \"$CLUSTER_IMAGE\", \"central_api_endpoint\": \"$CLUSTER_API_ENDPOINT\" $EXTRA_JSON}"

    TMP=$(mktemp)
    STATUS=$(curl -X POST \
        -d "$CLUSTER_JSON" \
        -k \
        -s \
        -o $TMP \
        -w "%{http_code}\n" \
        https://$LOCAL_API_ENDPOINT/v1/clusters)
    echo "Status: $STATUS"
    echo "Response: $(cat ${TMP})"
    cat "$TMP" | jq -r .deploymentYaml > "$OUTPUT_DIR/sensor-$CLUSTER_NAME-deploy.yaml"
    rm "$TMP"
    echo
}

# get_identity
# arguments:
#   - central API server endpoint reachable from this host
#   - name of cluster
#   - directory to drop files in
function get_identity {
    LOCAL_API_ENDPOINT="$1"
    CLUSTER_NAME="$2"
    OUTPUT_DIR="$3"

    echo "Getting identity for new cluster"
    export ID_JSON="{\"name\": \"$CLUSTER_NAME\", \"type\": \"SENSOR_SERVICE\"}"
    TMP=$(mktemp)
    STATUS=$(curl -X POST \
        -d "$ID_JSON" \
        -k \
        -s \
        -o "$TMP" \
        -w "%{http_code}\n" \
        https://$LOCAL_API_ENDPOINT/v1/serviceIdentities)
    echo "Status: $STATUS"
    echo "Response: $(cat ${TMP})"
    cat "$TMP" | jq -r .certificate > "$OUTPUT_DIR/sensor-$CLUSTER_NAME-cert.pem"
    cat "$TMP" | jq -r .privateKey > "$OUTPUT_DIR/sensor-$CLUSTER_NAME-key.pem"
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
    STATUS=$(curl \
        -k \
        -s \
        -o "$TMP" \
        -w "%{http_code}\n" \
        https://$LOCAL_API_ENDPOINT/v1/authorities)
    echo "Status: $STATUS"
    echo "Response: $(cat ${TMP})"
    cat "$TMP" | jq -r .authorities[0].certificate > "$OUTPUT_DIR/central-ca.pem"
    rm "$TMP"
    echo
}


