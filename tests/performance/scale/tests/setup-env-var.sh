#!/usr/bin/env bash
set -eoux pipefail

ELASTIC_USERNAME=$1
ELASTIC_PASSWORD=$2
INFRA_TOKEN=$3

export ELASTIC_USERNAME="$ELASTIC_USERNAME"
export ELASTIC_PASSWORD="$ELASTIC_PASSWORD"

export ELASTICSEARCH_URL=https://${ELASTIC_USERNAME}:${ELASTIC_PASSWORD}@search-acs-perfscale-koafbspz7ynsknj7r6cxxlmqh4.us-east-1.es.amazonaws.com

export INFRA_TOKEN="$INFRA_TOKEN"

export ARTIFACTS_DIR="${HOME}/artifacts"

export KUBECONFIG="${ARTIFACTS_DIR}/kubeconfig"

export PROMETHEUS_URL="https://$(oc get route --namespace openshift-monitoring prometheus-k8s --output jsonpath='{.spec.host}' | xargs)"

export PROMETHEUS_TOKEN="$(oc serviceaccounts new-token --namespace openshift-monitoring prometheus-k8s)"

source ./env.sh
