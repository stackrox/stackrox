#!/usr/bin/env bash
set -e

SWARM_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.mitigate_net:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"

FLAGS=""
if [ "$REGISTRY_AUTH" = "true" ]; then
    FLAGS="--with-registry-auth"
fi

generate_ca "$SWARM_DIR"

echo "Deploying Central..."
WD="$(pwd)"
cd "$SWARM_DIR"
docker stack deploy -c "$SWARM_DIR/central.yaml" mitigate $FLAGS
cd "$WD"
echo

wait_for_central "$LOCAL_API_ENDPOINT"
CLUSTER="remote"
create_cluster "$LOCAL_API_ENDPOINT" "$CLUSTER" SWARM_CLUSTER "$MITIGATE_IMAGE" "$CLUSTER_API_ENDPOINT" "$SWARM_DIR"
get_identity "$LOCAL_API_ENDPOINT" "$CLUSTER" "$SWARM_DIR"
get_authority "$LOCAL_API_ENDPOINT" "$SWARM_DIR"

echo "Deploying Sensor..."
if [ "$FLAGS" != "" ]; then
    SCRIPT_TMP=$(mktemp)
    chmod +x $SCRIPT_TMP
    cat "$SWARM_DIR/sensor-deploy.sh" | sed "s/stack deploy -c/stack deploy $FLAGS -c/" > $SCRIPT_TMP
    $SCRIPT_TMP
else
    $SWARM_DIR/sensor-deploy.sh
fi
echo

echo "Successfully deployed!"
echo "Access the UI at: https://$LOCAL_API_ENDPOINT"
