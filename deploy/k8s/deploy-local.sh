#!/usr/bin/env bash

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# assuming deployment on Kubernetes in Docker for Mac or Minikube
export COLLECTION_METHOD="${COLLECTION_METHOD:-ebpf}"
export MONITORING_SUPPORT="${MONITORING_SUPPORT:-false}"
export POD_SECURITY_POLICIES="${POD_SECURITY_POLICIES:-false}"

# shellcheck source=/dev/null
"$DIR"/deploy.sh
