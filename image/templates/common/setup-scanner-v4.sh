#!/usr/bin/env bash

# Sets up the Central deployment to use Scanner V4

# shellcheck disable=SC1083
KUBE_COMMAND=${KUBE_COMMAND:-{{.K8sConfig.Command}}}
NAMESPACE="${ROX_NAMESPACE:-stackrox}"

${KUBE_COMMAND} -n "$NAMESPACE" set env deploy/central -c="central" ROX_SCANNER_V4=true
