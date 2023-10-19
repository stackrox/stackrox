# A library of bash functions useful for installing operator using OLM.

declare -r KUTTL="${KUTTL:-kubectl-kuttl}"
declare -r pull_secret="operator-pull-secret"
declare -r USE_MIDSTREAM_IMAGES=${USE_MIDSTREAM_IMAGES:-false}
# `declare` ignores `errexit`: http://mywiki.wooledge.org/BashFAQ/105
ROOT_DIR="$(dirname "${BASH_SOURCE[0]}")/../.."
readonly ROOT_DIR

function log() {
    echo "$(date -u "+%Y-%m-%dT%H:%M:%SZ")" "$@" >&2
}

function create_namespace() {
  local -r operator_ns="$1"
  log "Creating namespace..."
  echo '{"kind": "Namespace", "apiVersion": "v1", "metadata": { "name": "'"${operator_ns}"'" } }' \
    | kubectl apply -f -
}

function create_pull_secret() {
  local -r operator_ns="$1"
  local -r registry_hostname="$2"
  # Note: can get rid of this secret once its in the cluster global pull secrets,
  # see https://stack-rox.atlassian.net/browse/RS-261
  log "Creating image pull secret..."
  "${ROOT_DIR}/deploy/common/pull-secret.sh" "${pull_secret}" "${registry_hostname}" \
    | kubectl -n "${operator_ns}" apply -f -
}

function apply_operator_manifests() {
  log "Applying operator manifests..."
  local -r operator_ns="$1"
  local -r image_tag_base="$2"
  local -r index_version="$3"
  local -r operator_version="$4"

  # OCP starting from v4.14 requires either spec.grpcPodConfig.securityContextConfig attribute to be set on the
  # CatalogSource resource or the namespace of the CatalogSource to have relaxed PSA enforcement, otherwise the
  # CatalogSource pod does not get deployed with PodSecurity errors.
  # Here we check if the CatalogSource CRD has securityContextConfig attribute. When it's not available, we emit YAML
  # comment symbol "# " to hide the securityContextConfig attribute and not get API validation errors.
  local catalog_source_crd
  catalog_source_crd="$(kubectl get customresourcedefinitions.apiextensions.k8s.io catalogsources.operators.coreos.com -o yaml)"
  local disable_security_context_config="# "
  local has_scc_key
  has_scc_key="$(yq eval '[.. | select(key == "grpcPodConfig")].[].properties | has("securityContextConfig")' - <<< "$catalog_source_crd")"
  if [[ "$has_scc_key" == "true" ]]; then
      disable_security_context_config=""
  fi

  if [[ "${USE_MIDSTREAM_IMAGES}" == "true" ]]; then
    # Get Operator channel from json for midstream
    operator_channel=$(< midstream/iib.json jq -r '.operator.channel')
  env -i PATH="${PATH}" \
    INDEX_VERSION="${index_version}" OPERATOR_VERSION="${operator_version}" NAMESPACE="${operator_ns}" OPERATOR_CHANNEL="${operator_channel}" \
    IMAGE_TAG_BASE="${image_tag_base}" \
    envsubst < "${ROOT_DIR}/operator/hack/operator-midstream.envsubst.yaml" \
    | kubectl -n "${operator_ns}" apply -f -
  else
  env -i PATH="${PATH}" \
    INDEX_VERSION="${index_version}" OPERATOR_VERSION="${operator_version}" NAMESPACE="${operator_ns}" \
    IMAGE_TAG_BASE="${image_tag_base}" DISABLE_SECURITY_CONTEXT_CONFIG="${disable_security_context_config}" \
    envsubst < "${ROOT_DIR}/operator/hack/operator.envsubst.yaml" \
    | kubectl -n "${operator_ns}" apply -f -
  fi
}

function retry() {
  local -ir max_attempts=$1; shift
  local -ir sleep=$1; shift
  for attempt in $(seq "${max_attempts}")
  do
    if (( attempt > 1 )); then
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

function check_version_tag() {
  local -r version_tag="$1"
  local -r allow_dirty_tag="$2"

  if [[ "$version_tag" == *-dirty ]]; then
    log "Target image tag has -dirty suffix."
    if [[ "$allow_dirty_tag" == false ]]; then
      log "Cannot install from *-dirty image tag. Please, use 'deploy-dirty-tag-via-olm' command or add '--allow-dirty-tag' flag if you need to install dirty tagged image."
      return 1
    fi
  fi

  return 0
}

function approve_install_plan() {
  local -r operator_ns="$1"
  local -r version_tag="$2"

  log "Waiting for an install plan to be created"
  if ! retry 10 5 kubectl -n "${operator_ns}" wait subscription.operators.coreos.com stackrox-operator-test-subscription --for condition=InstallPlanPending --timeout=60s; then
    log "Install plan failed to materialize."
    log "Dumping pod descriptions..."
    kubectl -n "${operator_ns}" describe pods -l "olm.catalogSource=stackrox-operator-test-index" || true
    log "Dumping catalog sources and subscriptions..."
    kubectl -n "${operator_ns}" describe "subscription.operators.coreos.com,catalogsource.operators.coreos.com" || true
    return 1
  fi

  log "Verifying that the subscription is progressing to the expected CSV of ${version_tag}..."
  local current_csv
  # `local` ignores `errexit` so we assign value separately: http://mywiki.wooledge.org/BashFAQ/105
  current_csv=$(kubectl get -n "${operator_ns}" subscription.operators.coreos.com stackrox-operator-test-subscription -o jsonpath="{.status.currentCSV}")
  readonly current_csv
  local -r expected_csv="rhacs-operator.v${version_tag}"
  if [[ $current_csv != $expected_csv ]]; then
    log "Subscription is progressing to unexpected CSV '${current_csv}', expected '${expected_csv}'"
    return 1
  fi

  local install_plan_name
  # `local` ignores `errexit` so we assign value separately: http://mywiki.wooledge.org/BashFAQ/105
  install_plan_name=$(kubectl get -n "${operator_ns}" subscription.operators.coreos.com stackrox-operator-test-subscription -o jsonpath="{.status.installPlanRef.name}")
  readonly install_plan_name

  log "Approving install plan ${install_plan_name}"
  retry 3 5 kubectl -n "${operator_ns}" patch installplan "${install_plan_name}" --type merge -p '{"spec":{"approved":true}}'
}

function nurse_deployment_until_available() {
  local -r operator_ns="$1"
  local -r version_tag="$2"

  log "Patching image pull secret into ${version_tag} CSV..."
  retry 30 10 kubectl -n "${operator_ns}" patch clusterserviceversions.operators.coreos.com \
    "rhacs-operator.v${version_tag}" --type json \
    -p '[ { "op": "add", "path": "/spec/install/spec/deployments/0/spec/template/spec/imagePullSecrets", "value": [{"name": "'"${pull_secret}"'"}] } ]'

  # Just waiting turns out to be the quickest and most reliable way of propagating the change.
  # Deleting the deployment sometimes tends to never get reconciled, with evidence of the
  # reconciliation failing with "not found" errors. OTOH simply leaving an unhealthy deployment around
  # means it will get updated eventually (and usually in under a minute).

  # We check the CSV status first, because it is hard to wait for the deployment in a non-racy way:
  # the deployment .status is set separately from the .spec, so the .status reflects the status of
  # the _old_ .spec until the deployment controller runs the first reconciliation.
  # We use kuttl because CSV has a Condition type incompatible with `kubectl wait`.
  log "Waiting for the ${version_tag} CSV to finish installing."
  "${KUTTL}" assert --timeout 300 --namespace "${operator_ns}" /dev/stdin <<-END
apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: rhacs-operator.v${version_tag}
status:
  phase: Succeeded
END

  # Double-check that the deployment itself is healthy.
  log "Making sure the ${version_tag} operator deployment is available..."
  retry 3 5 kubectl -n "${operator_ns}" wait deployments.apps -l "olm.owner=rhacs-operator.v${version_tag}" --for condition=available --timeout 5s
}
