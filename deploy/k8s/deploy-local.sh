#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

# assuming deployment on Kubernetes in Docker for Mac or Minikube
export COLLECTION_METHOD="${COLLECTION_METHOD:-kernel-module}"
export MONITORING_SUPPORT="${MONITORING_SUPPORT:-false}"

export ROX_IMAGE_FLAVOR="${ROX_IMAGE_FLAVOR:-development_development}"

$DIR/deploy.sh
