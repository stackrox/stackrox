#!/usr/bin/env bash
# shellcheck disable=SC2086
# shellcheck disable=SC1091
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"


source $COMMON_DIR/env.sh
source $COMMON_DIR/deploy.sh
source $COMMON_DIR/k8sbased.sh
source $K8S_DIR/env.sh

launch_central "$K8S_DIR"
