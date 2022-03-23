#!/usr/bin/env bash

set -e

# ensure logged in to docker
docker login

# get clean cluster name
CLUSTER_NAME="$(echo $1 | sed 's/-rg//')"
if [[ -z "${CLUSTER_NAME}" ]]; then
  if [[ -n "$ROX_SETUP_ID" ]]; then
    CLUSTER_NAME="setup-${ROX_SETUP_ID}"
  else
    echo >&2 'Please specify a cluster name!'
    exit 1
  fi
fi

zone="--zone us-central1-a"
if [[ -n $ZONE ]]; then
    zone="--zone $ZONE"
fi
# Zone values for different types of infra clusters
# GKE (default): us-central1-a
# Demo: us-central1-b
# Openshift: us-east1-d

# get gcloud user name
gcloud container clusters get-credentials "${CLUSTER_NAME}" --project srox-temp-dev-test $zone || {
  exit 1
}
[[ -n "$GCLOUD_USER" ]] || GCLOUD_USER="$(gcloud config get-value account 2>/dev/null)"
[[ -n "$GCLOUD_USER" ]] || {
    echo >&2 'Please specify a gcloud username via the GCLOUD_USER environment variable'
    exit 1
}

# create cluster role bindings
role_binding_name="temporary-admin-$(echo "${GCLOUD_USER}" | sed 's/@.*//')"
kubectl get clusterrolebinding "${role_binding_name}" >/dev/null 2>&1 || kubectl create clusterrolebinding "${role_binding_name}" --clusterrole=cluster-admin --user="$GCLOUD_USER"
