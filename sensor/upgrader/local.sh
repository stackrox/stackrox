#!/bin/bash

BUNDLE_DIR="${HOME}/Downloads/sensor-legacy2"

# Workflows are sets of instructions to execute. Central sends those instructions when `-workflow` param is missng.
# The workflow "cleanup" will kill any deployment given in ROX_UPGRADER_OWNER

ROX_CLUSTER_ID=d7177dd3-d260-41f4-8e4a-c4d089546bc7 \
ROX_UPGRADER_OWNER="Deployment:apps/v1:stackrox/scanner-db" \
    ROX_MTLS_CERT_FILE="${BUNDLE_DIR}/sensor-cert.pem" \
    ROX_MTLS_KEY_FILE="${BUNDLE_DIR}/sensor-key.pem" \
    ROX_MTLS_CA_FILE="${BUNDLE_DIR}/ca.pem" \
    ROX_CENTRAL_ENDPOINT=localhost:8443 \
    go run ./... -kube-config kubectl -workflow roll-forward

# main: 2024/07/18 11:37:08.779580 main.go:36: Info: Running StackRox Version: 4.6.x-86-g7b0a445168
