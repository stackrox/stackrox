#!/bin/bash

BUNDLE_DIR="${HOME}/Downloads/sensor-legacy2"

ROX_CLUSTER_ID=d7177dd3-d260-41f4-8e4a-c4d089546bc7 \
    ROX_MTLS_CERT_FILE="${BUNDLE_DIR}/sensor-cert.pem" \
    ROX_MTLS_KEY_FILE="${BUNDLE_DIR}/sensor-key.pem" \
    ROX_MTLS_CA_FILE="${BUNDLE_DIR}/ca.pem" \
    ROX_MTLS_CA_KEY_FILE="${BUNDLE_DIR}/unknown.pem" \
    ROX_CENTRAL_ENDPOINT=localhost:8443 \
    go run ./... -kube-config kubectl -workflow roll-forward
