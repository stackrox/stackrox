#!/usr/bin/env bash
set -eo pipefail

# Uninstalls secured cluster services and deletes the dev-cluster from central

if [[ ! -x "$(command -v "roxhelp")" ]]; then
      echo >&2 echo "Could not find github.com/stackrox/workflow commands. Is it installed?"
      exit 1
fi

helm -n stackrox uninstall stackrox-secured-cluster-services || true
cluster_id="$(roxcurl /v1/clusters | jq -r '.clusters[] | select(.name == "dev-cluster") | .id')"
roxcurl "/v1/clusters/${cluster_id}" -XDELETE || true
echo ""
echo "Removed ${cluster_id}"
