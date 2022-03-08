#!/usr/bin/env bash
set -eo pipefail

registry="docker.io"
user=$(echo $REGISTRY_USERNAME)
password=$(echo $REGISTRY_PASSWORD)

oc get secret/pull-secret -n openshift-config --template='{{index .data ".dockerconfigjson" | base64decode}}' >/tmp/pull_secret

oc registry login --registry="$registry" --auth-basic="$user:$password" --to=/tmp/pull_secret

oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson=/tmp/pull_secret

