#!/usr/bin/env bash
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source "$COMMON_DIR"/env.sh
source "$COMMON_DIR"/deploy.sh
source "$COMMON_DIR"/k8sbased.sh
source "$K8S_DIR"/env.sh

echo "Debug: before launch_central run: $- ${BASH_SUBSHELL}"
launch_central "$K8S_DIR"
echo "Debug: after launch_central run: $- ${BASH_SUBSHELL}"
