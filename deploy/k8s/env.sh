#!/usr/bin/env bash
set -e

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.stackrox:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"

export RUNTIME_SUPPORT=${RUNTIME_SUPPORT:-false}
echo "RUNTIME_SUPPORT set to $RUNTIME_SUPPORT"

export MONITORING_SUPPORT=${MONITORING_SUPPORT:-true}
echo "MONITORING_SUPPORT set to ${MONITORING_SUPPORT}"

export CLUSTER=${CLUSTER:-remote}
echo "CLUSTER set to $CLUSTER"

