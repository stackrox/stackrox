#!/usr/bin/env bash

# This script demonstrates how to run a particular e2e test from console

set -ueE

# get that from Bitwarden
#export GOOGLE_CREDENTIALS_GCR_SCANNER='<get from bitwarden>'
#export REGISTRY_PASSWORD=''
#export REGISTRY_USERNAME='rhacs-eng+stackroxciro'

export CLUSTER=K8S
export ROX_PASSWORD="<pass to your cluster>"
export ROX_USERNAME='admin'
export KUBECONFIG="${HOME}/.kube/config"

./gradlew test --tests=NodeInventoryTest
