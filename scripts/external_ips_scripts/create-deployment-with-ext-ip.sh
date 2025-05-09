#!/usr/bin/env bash
set -eou pipefail

export TARGET_IP=$1
export PORT=$2
export NAME=$3

envsubst < external-destination-source-stable-template.yml > deployment.yml
kubectl apply -f deployment.yml
