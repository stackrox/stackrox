#!/usr/bin/env bash
set -e

export LOCAL_API_ENDPOINT="${LOCAL_API_ENDPOINT:-"localhost:8000"}"
echo "Local StackRox Prevent endpoint set to $LOCAL_API_ENDPOINT"

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.prevent_net:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"

export PREVENT_IMAGE_TAG=${PREVENT_IMAGE_TAG:-$(git describe --tags --abbrev=10 --dirty)}
export PREVENT_IMAGE=${PREVENT_IMAGE:-stackrox/prevent:$PREVENT_IMAGE_TAG}
echo "PREVENT_IMAGE set to $PREVENT_IMAGE"