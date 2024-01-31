#!/usr/bin/env bash

# Sets up Central to use Scanner-v4. Note that deploying Scanner is a prerequisite to deploying Scanner-v4.

KUBE_COMMAND=${KUBE_COMMAND:-{{.K8sConfig.Command}}}
NAMESPACE="${ROX_NAMESPACE:-stackrox}"

${KUBE_COMMAND} -n "$NAMESPACE" patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","env":[{"name": "ROX_SCANNER_V4", "value": "true"}]}]}}}}'
