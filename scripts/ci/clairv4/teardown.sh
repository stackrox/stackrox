#!/usr/bin/env bash

set -e

usage() {
    echo "usage: ./teardown.sh <namespace>"
}

if [ $# -lt 1 ]; then
  usage
  exit 1
fi

namespace="$1"

if kubectl get ns "${namespace}"; then
    kubectl delete ns "${namespace}"
fi
