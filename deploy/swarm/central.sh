#!/usr/bin/env bash
set -e

SWARM_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $SWARM_DIR/launch.sh
source $SWARM_DIR/env.sh

read -p "Review the above variables and hit enter to continue: "

launch_central "$ROX_CENTRAL_DASHBOARD_PORT" "$SWARM_DIR" "$PREVENT_IMAGE" "$LOCAL_API_ENDPOINT" "$PREVENT_DISABLE_REGISTRY_AUTH"
