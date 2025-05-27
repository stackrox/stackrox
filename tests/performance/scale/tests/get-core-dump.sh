#!/usr/bin/env bash
set -eou pipefail

CENTRAL_IP="$(kubectl -n stackrox get routes central -o json | jq -r '.spec.host')"
ROX_ADMIN_PASSWORD="$(cat $ARTIFACTS_DIR/kubeadmin-password)"

roxctl central debug dump -e ${CENTRAL_IP}:443 -p ${ROX_ADMIN_PASSWORD} --insecure-skip-tls-verify

# Alternatively
#kubectl -n stackrox port-forward svc/central 8443:443 &
#roxctl central debug dump -e ${CENTRAL_IP}:8443 -p ${ROX_ADMIN_PASSWORD} --insecure-skip-tls-verify
