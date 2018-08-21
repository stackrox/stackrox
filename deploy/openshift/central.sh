#!/usr/bin/env bash
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $K8S_DIR/launch.sh
source $K8S_DIR/env.sh

launch_central "$LOCAL_API_ENDPOINT" "$K8S_DIR" docker-registry.default.svc:5000 "$PREVENT_IMAGE_TAG"
