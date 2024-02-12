#!/usr/bin/env bash
set -eo pipefail

# Extract compliance operator CRDs
kubectl get crd | awk '{print $1}' | grep compliance.openshift.io | xargs kubectl get crd -o yaml > compliance-operator-crds.yaml


