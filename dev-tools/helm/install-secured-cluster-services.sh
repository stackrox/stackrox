#!/usr/bin/env bash
set -eo pipefail

# Install central services based on the new helm charts
# REQUIRED: running central deployment, see install-central-services.sh.
#
# Usage:
#
# $ cdrox # change to rox root directory
# $ roxctl helm output secured-cluster-services # generate helm chart
# $ kubectl -n stackrox svc/central 8000:443 & # port forward central to port 8000
# $ ./dev-tools/helm/install-secured-cluster-services.sh

SCRIPT="$(python -c 'import os, sys; print(os.path.realpath(sys.argv[1]))' "${BASH_SOURCE[0]}")"
source "$(dirname "$SCRIPT")/common-vars.sh"

cluster_id="$(roxcurl /v1/clusters | jq -r '.clusters[] | select(.name == "dev-cluster") | .id')"
[ -n "$cluster_id" ] && { echo >&2 "Cluster with name 'dev-cluster' already exists with ID '$cluster_id'"; exit 1; }

if [[ -z "${ROX_API_TOKEN}" ]]; then
  timestamp=$(date +%s)
  token_req=$(cat <<- EOF
{"name": "install-secured-cluster-services-token-${timestamp}", "roles": ["Admin"]}
EOF)

  echo "Create api token: install-secured-cluster-services-token-${timestamp}"
  export ROX_API_TOKEN="$(roxcurl /v1/apitokens/generate -XPOST -d "${token_req}" | jq -r .token)"
fi

export ROX_NO_IMAGE_PULL_SECRETS=true

./stackrox-secured-cluster-services-chart/scripts/setup.sh \
  -f ./dev-tools/helm/secured-cluster-services/docker-values-public.yaml \
  -e localhost:8000

helm upgrade --install -n stackrox --create-namespace stackrox-secured-cluster-services ./stackrox-secured-cluster-services-chart \
  -f ./dev-tools/helm/secured-cluster-services/docker-values-public.yaml \
  --set image.tag.main="${MAIN_IMAGE_TAG}"

echo "Deployed image: ${MAIN_IMAGE_TAG}"
