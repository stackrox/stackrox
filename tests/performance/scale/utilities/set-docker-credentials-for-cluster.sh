#!/usr/bin/env bash
set -eoux pipefail

if command -v docker-credential-osxkeychain || command -v docker-credential-secretservice; then
	export QUAY_REGISTRY_AUTH_BASIC=$($(command -v docker-credential-osxkeychain || command -v docker-credential-secretservice) get <<<"https://quay.io" | jq -r '"\(.Username):\(.Secret)"')
else
	export QUAY_REGISTRY_AUTH_BASIC="$DOCKER_USERNAME:$DOCKER_PASSWORD"
fi

oc get secret/pull-secret -n openshift-config --template='{{index .data ".dockerconfigjson" | base64decode}}' > ./tmp-pull-secret.json
oc registry login --registry="quay.io/rhacs-eng" --auth-basic="${QUAY_REGISTRY_AUTH_BASIC}" --to=./tmp-pull-secret.json
oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson=./tmp-pull-secret.json
