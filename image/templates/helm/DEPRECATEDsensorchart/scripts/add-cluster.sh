#!/bin/bash

if ! type jq >/dev/null 2>&1; then
	echo "Error: jq not found on your system. Please install jq and rerun this script." 1>&2
	exit 1
fi

if [ "$#" -ne 1 ]; then
    echo "error: central endpoint not specified, use -e with ./setup.sh"
    exit 1
fi

endpoint="${1}"
KUBE_COMMAND="${KUBE_COMMAND:-kubectl}"

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

. "${DIR}"/config.sh

SECRETS_DIR="${DIR}/../secrets"
mkdir -p "${SECRETS_DIR}"

if [ -z "${ROX_API_TOKEN}" ]; then
  echo "A valid API token must be provided" && exit 1
fi

"${KUBE_COMMAND}" get namespace stackrox &>/dev/null || ${KUBE_COMMAND} create namespace stackrox

#setup docker auth for image registries
if ! "${KUBE_COMMAND}" get secret/stackrox -n stackrox &>/dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "${image_registry_main}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  "${KUBE_COMMAND}" create --namespace "stackrox" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: stackrox
  namespace: stackrox
type: kubernetes.io/dockerconfigjson
EOF
fi

if [ "${config_collectionMethod}" != "NO_COLLECTION" ]; then
  if ! ${KUBE_COMMAND} get secret/collector-stackrox -n stackrox &>/dev/null; then
    registry_auth="$("${DIR}/docker-auth.sh" -m k8s "${image_registry_collector}")"
    [[ -n "${registry_auth}" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
    ${KUBE_COMMAND} create --namespace "stackrox" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: collector-stackrox
  namespace: stackrox
type: kubernetes.io/dockerconfigjson
EOF
  fi
fi

# add cluster
 cluster=$(jq -r -n \
  --arg cn "${cluster_name}"  \
  --arg ct "${cluster_type}"  \
  --arg mimage "${image_registry_main}/${image_repository_main}" \
  --arg cimage "${image_registry_collector}/${image_repository_collector}" \
  --arg ce "${endpoint_central}" \
  --arg cm "${config_collectionMethod}" \
  --argjson ac "${config_admissionControl_createService}" \
  --argjson acu "${config_admissionControl_listenOnUpdates}" \
  --argjson aes "${config_admissionControl_enableService}" \
  --argjson aeu "${config_admissionControl_enforceOnUpdates}" \
  --arg to "${config_admissionControl_timeout}" \
  --argjson aes "${config_admissionControl_enableService}" \
  --argjson si "${config_admissionControl_scanInline}" \
  --argjson db "${config_admissionControl_disableBypass}" \
  --arg ro "${config_registryOverride}" \
  --argjson td "${config_disableTaintTolerations}" \
  '{"cluster": { "name": $cn, "type": $ct, "mainImage": $mimage, "collectorImage": $cimage,
  "centralApiEndpoint": $ce, "collectionMethod": $cm, "admissionController": $ac,
  "admissionControllerUpdates": $acu, "dynamicConfig": { "admissionControllerConfig": { "enabled": $aes,
  "timeoutSeconds": $to, "scanInline": $si, "enforceOnUpdates": $aeu,
  "disableBypass": $db }, "registryOverride": $ro }, "tolerationsConfig": {"disabled": $td} } }')

  auth_header="Authorization: Bearer ${ROX_API_TOKEN}"
  certsZipFile="${DIR}/certs-${cluster_name}.zip"

 # add cluster, get certs bundle
  status="$(curl --insecure --output /dev/stderr --write-out "%{http_code}" --show-error -sKOJ -H "${auth_header}" -H "Accept-Encoding: zip" -H "Content-Type: application/json" \
  -X POST --data "${cluster}" "https://${endpoint}/api/helm/cluster/add" 2>"${certsZipFile}")"
  if [[ "$status" != 200 ]]; then
    cat "${certsZipFile}"
    exit 1
  fi

  if [ ! -f "${certsZipFile}" ]; then
    echo "Error: ${certsZipFile} not found."
    exit 1
  fi

  unzip "${certsZipFile}" -d "${SECRETS_DIR}"

  #clean up
  rm -f "${certsZipFile}"
  rm -f "${DIR}/config.sh"
