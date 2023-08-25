#!/usr/bin/env bash
# shellcheck disable=SC1091
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

# shellcheck source=../common/env.sh
source "$COMMON_DIR"/env.sh
# shellcheck source=../common/deploy.sh
source "$COMMON_DIR"/deploy.sh
# shellcheck source=../common/k8sbased.sh
source "$COMMON_DIR"/k8sbased.sh
# shellcheck source=./env.sh
source "$K8S_DIR"/env.sh

launch_central "$K8S_DIR"
