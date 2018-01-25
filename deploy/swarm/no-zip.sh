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
if [ "$MITIGATE_DISABLE_REGISTRY_AUTH" = "true" ]; then
    docker stack deploy -c "$SWARM_DIR/central.yaml" mitigate
else
    docker stack deploy -c "$SWARM_DIR/central.yaml" mitigate --with-registry-auth
fi
cd "$WD"
echo

wait_for_central "$LOCAL_API_ENDPOINT"
CLUSTER="remote"
CLUSTER_ID=$(create_cluster "$LOCAL_API_ENDPOINT" "$CLUSTER" SWARM_CLUSTER "$MITIGATE_IMAGE" "$CLUSTER_API_ENDPOINT" "$SWARM_DIR")
echo "Cluster ID: $CLUSTER_ID"
get_identity "$LOCAL_API_ENDPOINT" "$CLUSTER_ID" "$SWARM_DIR"
get_authority "$LOCAL_API_ENDPOINT" "$SWARM_DIR"

echo "Deploying Sensor..."
if [ "$MITIGATE_DISABLE_REGISTRY_AUTH" = "true" ]; then
    echo "Disabling registry auth in deployment..."
    cp "$SWARM_DIR/sensor-deploy.sh" "$SWARM_DIR/tmp"
    cat "$SWARM_DIR/tmp" | sed "s/--with-registry-auth//" > "$SWARM_DIR/sensor-deploy.sh"
    rm "$SWARM_DIR/tmp"
fi

$SWARM_DIR/sensor-deploy.sh
echo

echo "Successfully deployed!"
echo "Access the UI at: https://$LOCAL_API_ENDPOINT"
