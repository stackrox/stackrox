#!/bin/bash
# Installs the operator using OLM which is already running in the cluster.
set -eu -o pipefail

root_dir="$(dirname "$0")/../.."
pull_secret="operator-pull-secret"
image_tag_base="${IMAGE_TAG_BASE:-docker.io/stackrox/stackrox-operator}"
registry_hostname="${image_tag_base%%/*}"
image_registry="${image_tag_base%/*}"

operator_ns="${1:-}"
version_tag="${2:-}"
if [[ -z ${operator_ns} || -z ${version_tag} ]]; then
  echo "Usage: $0 <operator_ns> <version-tag>" >&2
  exit 1
fi

function log() {
    echo "$(date -u "+%Y-%m-%dT%H:%M:%SZ")" "$@" >&2
}

log "Creating namespace..."
echo "{'kind': 'Namespace', 'apiVersion': 'v1', 'metadata': { 'name': '${operator_ns}' } }" | tr \' \" |
 kubectl apply -f -

log "Creating image pull secret..."
"${root_dir}/deploy/common/pull-secret.sh" "${pull_secret}" "${registry_hostname}" |
 kubectl -n "${operator_ns}" apply -f -

log "Applying operator manifests..."
env - PATH="${PATH}" \
  VERSION_TAG="${version_tag}" NAMESPACE="${operator_ns}" \
  `# TODO(ROX-7740): Remove the following two once we have a single dev+CI repo.` \
  IMAGE_TAG_BASE="${image_tag_base}" IMAGE_REGISTRY="${image_registry}" \
  envsubst < "${root_dir}/operator/hack/operator.envsubst.yaml" |
  kubectl -n "${operator_ns}" apply -f -

function retry() {
  max_attempts=$1; shift
  sleep=$1; shift
  for attempt in $(seq "${max_attempts}")
  do
    if (( attempt > 1 ))
    then
      log "retrying in ${sleep}s..."
      sleep "${sleep}"
    fi
    if "$@"
    then
      return 0
    fi
  done
  log "failed $max_attempts attempts"
  return 1
}

log "Patching image pull secret into CSV..."
retry 20 5 kubectl -n "${operator_ns}" patch clusterserviceversions.operators.coreos.com \
    "rhacs-operator.v${version_tag}" --type json \
    -p "[ { \"op\": \"add\", \"path\": \"/spec/install/spec/deployments/0/spec/template/spec/imagePullSecrets\", \"value\": [{\"name\": \"${pull_secret}\"}] } ]"

# Just waiting turns out to be the quickest and most reliable way of propagating the change.
# Deleting the deployment sometimes tends to never get reconciled, with evidence of the
# reconciliation failing with "not found" errors. OTOH simply leaving an unhealthy deployment around
# means it will get updated eventually (and usually in under a minute).
log "Waiting for operator deployment to become available..."
retry 10 60 kubectl -n "${operator_ns}" wait deployments.apps/rhacs-operator-controller-manager --for condition=available --timeout 60s
