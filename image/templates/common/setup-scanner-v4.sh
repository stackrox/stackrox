#!/usr/bin/env bash

# Sets up the Central deployment to use Scanner V4

# shellcheck disable=SC1083
KUBE_COMMAND=${KUBE_COMMAND:-{{.K8sConfig.Command}}}
NAMESPACE="${ROX_NAMESPACE:-stackrox}"

${KUBE_COMMAND} -n "$NAMESPACE" patch deploy/central --patch-file <(cat <<EOF
spec:
  template:
    spec:
      containers:
      - name: central
        env:
        - name: "ROX_SCANNER_V4"
          value: "true"
EOF
)
