#!/usr/bin/env bash
set -e

SWARM_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.prevent_net:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"

export PREVENT_IMAGE="stackrox/prevent:${PREVENT_IMAGE_TAG:-latest}"

echo "Generating central config..."
OLD_DOCKER_HOST="$DOCKER_HOST"
OLD_DOCKER_CERT_PATH="$DOCKER_CERT_PATH"
OLD_DOCKER_TLS_VERIFY="$DOCKER_TLS_VERIFY"
unset DOCKER_HOST DOCKER_CERT_PATH DOCKER_TLS_VERIFY

docker run "$PREVENT_IMAGE" -t swarm -i "$PREVENT_IMAGE" -p 8080 > $SWARM_DIR/central.zip

export DOCKER_HOST="$OLD_DOCKER_HOST"
export DOCKER_CERT_PATH="$OLD_DOCKER_CERT_PATH"
export DOCKER_TLS_VERIFY="$OLD_DOCKER_TLS_VERIFY"

UNZIP_DIR="$SWARM_DIR/central-deploy/"
rm -rf "$UNZIP_DIR"
unzip "$SWARM_DIR/central.zip" -d "$UNZIP_DIR"
echo

echo "Deploying Central..."
if [ "$PREVENT_DISABLE_REGISTRY_AUTH" = "true" ]; then
    cp "$UNZIP_DIR/deploy.sh" "$UNZIP_DIR/tmp"
    cat "$UNZIP_DIR/tmp" | sed "s/--with-registry-auth//" > "$UNZIP_DIR/deploy.sh"
    rm "$UNZIP_DIR/tmp"
fi
$UNZIP_DIR/deploy.sh
echo

wait_for_central "$LOCAL_API_ENDPOINT"
EXTRA_CONFIG=""
if [ "$DOCKER_CERT_PATH" = "" ]; then
    EXTRA_CONFIG="\"disableSwarmTls\":true"
fi
CLUSTER="remote"
CLUSTER_ID=$(create_cluster "$LOCAL_API_ENDPOINT" "$CLUSTER" SWARM_CLUSTER "$PREVENT_IMAGE" "$CLUSTER_API_ENDPOINT" "$SWARM_DIR" "$EXTRA_CONFIG")
echo "Cluster ID: $CLUSTER_ID"
get_identity "$LOCAL_API_ENDPOINT" "$CLUSTER_ID" "$SWARM_DIR"
get_authority "$LOCAL_API_ENDPOINT" "$SWARM_DIR"

echo "Deploying Sensor..."
if [ "$PREVENT_DISABLE_REGISTRY_AUTH" = "true" ]; then
    echo "Disabling registry auth in deployment..."
    cp "$SWARM_DIR/sensor-deploy.sh" "$SWARM_DIR/tmp"
    cat "$SWARM_DIR/tmp" | sed "s/--with-registry-auth//" > "$SWARM_DIR/sensor-deploy.sh"
    rm "$SWARM_DIR/tmp"
fi

$SWARM_DIR/sensor-deploy.sh
echo

echo "Successfully deployed!"
echo "Access the UI at: https://$LOCAL_API_ENDPOINT"
