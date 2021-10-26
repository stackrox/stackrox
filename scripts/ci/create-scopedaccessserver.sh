#!/usr/bin/env bash

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

PLUGIN_VERSION="1.1"

tmpdir="$(mktemp -d)"
gsutil cat "gs://sr-authz-plugin-src/${PLUGIN_VERSION}/default-authz-plugin-${PLUGIN_VERSION}-src.tar.gz" \
	| tar -C "$tmpdir" -xvzf -

plugin_dir="${tmpdir}/default-authz-plugin"

export TLS_CERT_FILE="${DIR}/scopedaccess/config/server-tls.crt"
export TLS_KEY_FILE="${DIR}/scopedaccess/config/server-tls.key"

export SERVER_CONFIG_FILE="${DIR}/scopedaccess/config/server-config.json"
export RULES_FILE="${DIR}/scopedaccess/config/rules.gval"

QUAY_REPO="rhacs-eng"
export AUTHZ_PLUGIN_IMAGE="quay.io/$QUAY_REPO/default-authz-plugin:${PLUGIN_VERSION}"

"${plugin_dir}/examples/deployment/deploy.sh"

# patch the deployment to allow image pulls in the case where the secret named "stackrox" is
# a non-standard registry (i.e. something other then docker.io/stackrox or stackrox.io) and
# a secret named "stackrox-dockerhub" is needed to get access to the authorization-plugin image.
label_patch='{"op": "add", "path": "/spec/template/metadata/labels/imagepullsecret-added", "value": "true"}'
pull_secret_patch='{"op": "add", "path": "/spec/template/spec/imagePullSecrets/1", "value": {"name": "stackrox-dockerhub"}}'
kubectl -n stackrox patch deployment authorization-plugin --type='json' -p="[${label_patch},${pull_secret_patch}]"

sleep 5
POD=$(kubectl -n stackrox get po -lapp=authorization-plugin -limagepullsecret-added=true -o jsonpath='{.items[0].metadata.name}')
echo "$POD"
kubectl -n stackrox wait --for=condition=ready "pod/$POD" --timeout=2m
