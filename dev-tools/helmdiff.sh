#!/bin/bash

set -euo pipefail

DEV_TOOLS_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROXCTL="$DEV_TOOLS_ROOT/roxctl.sh"

CHARTS="central-services secured-cluster-services"
TMP_ROOT=$(mktemp -d -t helm_diff_)
WORKING_BRANCH=$(git branch --show-current)

echo "Generating temporary files in ${TMP_ROOT}"

echo "Rendering from metatemplates to helm charts:"
for CHART in ${CHARTS}; do
  $ROXCTL helm output --debug --remove "${CHART}" --output-dir="${TMP_ROOT}/${CHART}-new"
done

REPO_STAT="$(git diff --stat)"
# TODO(ebensh): Use smart branch root (if present) instead of master.
if [[ -n $REPO_STAT ]]; then
  echo "Saving uncommitted changes with 'git stash push'."
  git stash push
fi
git switch master
for CHART in ${CHARTS}; do
  $ROXCTL helm output --debug --remove "${CHART}" --output-dir="${TMP_ROOT}/${CHART}-old"
done
git switch "${WORKING_BRANCH}"
if [[ -n $REPO_STAT ]]; then
  echo "Restoring uncommitted changes with 'git stash pop'."
  git stash pop
fi

echo "Rendering a dry run installation of the stackrox-central-services helm charts as Kubernetes manifests:"
for VERSION in "old" "new"; do
  helm install --debug --dry-run \
    -n stackrox \
    --disable-openapi-validation \
    --set central.persistence.none=true \
    stackrox-central-services \
    "${TMP_ROOT}"/central-services-${VERSION} \
    > "${TMP_ROOT}"/central-services-${VERSION}-installation.yaml
done

echo "Rendering a dry run installation of the stackrox-secured-cluster-services helm charts as Kubernetes manifests:"
for VERSION in "old" "new"; do
  helm install --debug --dry-run \
    -n stackrox \
    --disable-openapi-validation \
    --set clusterName=mergers-and-acquisitions \
    --set ca.cert=a1 \
    --set serviceTLS.cert=b2 \
    --set serviceTLS.key=c3 \
    --set createSecrets=false \
    stackrox-secured-cluster-services \
    "${TMP_ROOT}"/secured-cluster-services-${VERSION} \
    > "${TMP_ROOT}"/secured-cluster-services-${VERSION}-installation.yaml
done

cat <<EOF

=== ${TMP_ROOT} ===

To compare helm charts:
diff -ruN "${TMP_ROOT}/central-services-old" "${TMP_ROOT}/central-services-new"
diff -ruN "${TMP_ROOT}/secured-cluster-services-old" "${TMP_ROOT}/secured-cluster-services-new"

To compare stackrox-central-services chart installation (Kubernetes manifests):"
diff -ruN \
  ${TMP_ROOT}/central-services-old-installation.yaml \
  ${TMP_ROOT}/central-services-new-installation.yaml

To compare secured-cluster-services chart installation (Kubernetes manifests):"
diff -ruN \
  ${TMP_ROOT}/secured-cluster-services-old-installation.yaml \
  ${TMP_ROOT}/secured-cluster-services-new-installation.yaml
EOF
