#!/usr/bin/env bash
set -e

SWARM_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-apollo.apollo_net:443}"
echo "In-cluster Apollo endpoint set to $CLUSTER_API_ENDPOINT"

FLAGS=""
if [ "$REGISTRY_AUTH" = "true" ]; then
    FLAGS="--with-registry-auth"
fi

generate_ca "$SWARM_DIR"

echo "Deploying Central..."
docker stack deploy -c "$SWARM_DIR/central.yaml" apollo $FLAGS
echo

wait_for_central "$LOCAL_API_ENDPOINT"
CLUSTER="remote"
create_cluster "$LOCAL_API_ENDPOINT" "$CLUSTER" SWARM_CLUSTER "$APOLLO_IMAGE" "$CLUSTER_API_ENDPOINT" "$SWARM_DIR"
get_identity "$LOCAL_API_ENDPOINT" "$CLUSTER" "$SWARM_DIR"
get_authority "$LOCAL_API_ENDPOINT" "$SWARM_DIR"

echo "Deploying Sensor..."
docker stack deploy -c "$SWARM_DIR/sensor-$CLUSTER_NAME-deploy.yaml" apollo $FLAGS
echo

echo "Successfully deployed!"
echo "Access the UI at: https://$LOCAL_API_ENDPOINT"
