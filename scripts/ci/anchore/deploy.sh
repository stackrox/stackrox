#!/usr/bin/env bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

usage() {
    echo "usage: ./deploy.sh <namespace> <anchore app name>"
}

if [ $# -lt 2 ]; then
  usage
  exit 1
fi

namespace="$1"
app_name="$2"

if [[ -z "${ANCHORE_PASSWORD}" || -z "${ANCHORE_USERNAME}" ]]; then
    echo "ANCHORE_PASSWORD and ANCHORE_USERNAME are both required."
    exit 1
fi

count_running_pods() {
    kubectl -n "${namespace}" get pods --field-selector=status.phase=="Running" -o json | jq -j '.items | length'
}

wait_for_anchore_to_start() {
    tries=0
    while [[ $(count_running_pods) -ne "6" ]]; do
        tries=$((tries + 1))
        if (( tries > 40 )); then
            kubectl get nodes -o yaml
            echo "Took too long to start"
            exit 1
        fi
        echo "Waiting for Anchore to start..."
        kubectl -n "${namespace}" get pods
        sleep 10
    done
    kubectl -n "${namespace}" get pods
}

wait_for_anchore_to_stop() {
    tries=0
    while [[ $(count_running_pods) -ne "1" ]]; do
        tries=$((tries + 1))
        if (( tries > 20 )); then
            echo "Took too long to stop"
            exit 1
        fi
        echo "Waiting for Anchore to stop..."
        kubectl -n "${namespace}" get pods
        sleep 10
    done
    kubectl -n "${namespace}" get pods
}

# Install

set -x
helm version
helm repo add anchore https://charts.anchore.io
helm repo update
if kubectl get ns "${namespace}"; then
    kubectl delete ns "${namespace}" # handle CI re-runs
fi
kubectl create ns "${namespace}"
kubectl -n "${namespace}" apply -f "${DIR}/psp.yaml"

set +e
helm install anchore/anchore-engine \
    -n "${app_name}" \
    --namespace "${namespace}" \
    --set "anchoreGlobal.defaultAdminPassword=${ANCHORE_PASSWORD}" \
    --set "postgresql.postgresPassword=${ANCHORE_PASSWORD}" \
    --set "postgresql.postgresUser=${ANCHORE_USERNAME}" \
    --version 1.7.0
if [[ $? -ne 0 ]]; then
    set -e
    # v3
    helm install "${app_name}" anchore/anchore-engine \
        --namespace "${namespace}" \
        --set "anchoreGlobal.defaultAdminPassword=${ANCHORE_PASSWORD}" \
        --set "postgresql.postgresPassword=${ANCHORE_PASSWORD}" \
        --set "postgresql.postgresUser=${ANCHORE_USERNAME}" \
        --version 1.7.0
fi
set -e

# Wait until all anchore pods are all running

wait_for_anchore_to_start

# Scale everything down except postgres

others="${app_name}-anchore-engine-analyzer \
${app_name}-anchore-engine-api \
${app_name}-anchore-engine-catalog \
${app_name}-anchore-engine-policy \
${app_name}-anchore-engine-simplequeue"

for deploy in ${others}; do
    kubectl -n "${namespace}" scale --replicas 0 deploy "${deploy}"
done

wait_for_anchore_to_stop

# Seed the DB with debian:8 vulns

kubectl -n "${namespace}" port-forward "svc/${app_name}-postgresql" 5432:5432 > /dev/null &
PID=$!

sleep 5

gsutil cat gs://stackrox-ci/anchore-1-6-9-debian8-vulns.sql | PGPASSWORD="${ANCHORE_PASSWORD}" psql -h localhost -U "${ANCHORE_USERNAME}" postgres 

kill ${PID}
wait

# Scale back up

for deploy in ${others}; do
    kubectl -n "${namespace}" scale --replicas 1 deploy "${deploy}"
done

wait_for_anchore_to_start

# More than a single analyzer is needed to bypass blocked analysis. See ROX-7807.
kubectl -n "${namespace}" scale --replicas 3 "deploy/${app_name}-anchore-engine-analyzer"

sleep 10 # required to let Anchore get ready

# See how we did

kubectl -n "${namespace}" port-forward "svc/${app_name}-anchore-engine-api" 8228:8228 > /dev/null &
PID=$!

sleep 5

set +e # ignore errors from here - anchore can be flaky on startup
export ANCHORE_CLI_USER="${ANCHORE_USERNAME}"
export ANCHORE_CLI_PASS="${ANCHORE_PASSWORD}"
export LC_ALL=C.UTF-8
anchore-cli system wait
anchore-cli system status
anchore-cli system feeds list

kill ${PID}
wait
