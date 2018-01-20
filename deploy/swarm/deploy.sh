#!/usr/bin/env bash
set -e

SWARM_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.mitigate_net:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"

generate_ca "$SWARM_DIR"

echo "Deploying Central..."
WD="$(pwd)"
cd "$SWARM_DIR"
if [ "$APOLLO_NO_REGISTRY_AUTH" = "true" ]; then
    docker stack deploy -c "$SWARM_DIR/central.yaml" mitigate
else
    docker stack deploy -c "$SWARM_DIR/central.yaml" mitigate --with-registry-auth
fi
cd "$WD"
echo

wait_for_central "$LOCAL_API_ENDPOINT"
CLUSTER="remote"
get_cluster_zip "$LOCAL_API_ENDPOINT" "$CLUSTER" SWARM_CLUSTER "$MITIGATE_IMAGE" "$CLUSTER_API_ENDPOINT" "$SWARM_DIR"

echo "Deploying Sensor..."
UNZIP_DIR="$SWARM_DIR/sensor-deploy/"
rm -rf "$UNZIP_DIR"
unzip "$SWARM_DIR/sensor-deploy.zip" -d "$UNZIP_DIR"

if [ "$APOLLO_NO_REGISTRY_AUTH" = "true" ]; then
    cp "$UNZIP_DIR/sensor-deploy.sh" "$UNZIP_DIR/tmp"
    cat "$UNZIP_DIR/tmp" | sed "s/--with-registry-auth//" > "$UNZIP_DIR/sensor-deploy.sh"
fi

$UNZIP_DIR/sensor-deploy.sh
echo

echo "Successfully deployed!"
echo "Access the UI at: https://$LOCAL_API_ENDPOINT"
