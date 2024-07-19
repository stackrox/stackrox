#!/bin/bash

BUNDLE_DIR="${HOME}/Downloads/sensor-legacy2"

# Workflows are sets of instructions to execute. Central sends those instructions when `-workflow` param is missng.
# The workflow "cleanup" will kill any deployment given in ROX_UPGRADER_OWNER

# The upgrader uses SA sensor. That SA is only for namespace stackrox

# When run on cluster:
# Environment:
# ROX_CENTRAL_ENDPOINT:    central.stackrox:443
# ROX_UPGRADE_PROCESS_ID:  1e2d5cad-af6a-4c2a-810e-62e40aca9459
# ROX_CLUSTER_ID:          d7177dd3-d260-41f4-8e4a-c4d089546bc7
# ROX_UPGRADER_OWNER:      Deployment:apps/v1:stackrox/sensor-upgrader

ROX_CLUSTER_ID=d7177dd3-d260-41f4-8e4a-c4d089546bc7 \
ROX_UPGRADER_OWNER="Deployment:apps/v1:stackrox/scanner-db" \
# The certs are provided to the upgrader through a volume
    ROX_MTLS_CERT_FILE="${BUNDLE_DIR}/sensor-cert.pem" \
    ROX_MTLS_KEY_FILE="${BUNDLE_DIR}/sensor-key.pem" \
    ROX_MTLS_CA_FILE="${BUNDLE_DIR}/ca.pem" \
    ROX_CENTRAL_ENDPOINT=localhost:8443 \
    go run ./... -kube-config kubectl -workflow dry-run

# main: 2024/07/18 11:37:08.779580 main.go:36: Info: Running StackRox Version: 4.6.x-86-g7b0a445168
#
# There was an error executing it: executing stage Run preflight checks:
# preflight check "Kubernetes authorization" reported errors:                                                                                                                                       │
# K8s authorizer did not explicitly allow or deny access to perform the following actions
# on following resources:
# update:rolebindings,
# update:clusterroles,
# update:clusterroles,
# update:clusterrolebindings,
# update:clusterroles,
# update:clusterroles,
# update:clusterroles,
# update:clusterroles,
# update:clusterrolebindings,
# update:prometheusrules,
# update:clusterroles,
# update:clusterroles,
# update:clusterrolebindings,
# update:clusterrolebindings,
# update:validatingwebhookconfigurations,
# update:servicemonitors,
# update:clusterrolebindings,
# update:clusterrolebindings,
# update:clusterroles,
# update:clusterroles,
# update:clusterrolebindings. This usually means access is denied.
