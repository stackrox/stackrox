# shellcheck shell=bash
# A library of bash functions useful for installing operator using OLM.

declare -r KUTTL="${KUTTL:-kubectl-kuttl}"
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
    | "${ROOT_DIR}/operator/hack/retry-kubectl.sh" apply -f -
}

function apply_operator_manifests() {
  log "Applying operator manifests..."
  local -r operator_ns="$1"
  local -r index_image_repo="$2"
  local -r index_image_tag="$3"
  local -r starting_csv_version="$4"
  local -r operator_channel="$5"

  # OCP starting from v4.14 requires either spec.grpcPodConfig.securityContextConfig attribute to be set on the
  # CatalogSource resource or the namespace of the CatalogSource to have relaxed PSA enforcement, otherwise the
  # CatalogSource pod does not get deployed with PodSecurity errors.
  # Here we check if the CatalogSource CRD has securityContextConfig attribute. When it's not available, we emit YAML
  # comment symbol "# " to hide the securityContextConfig attribute and not get API validation errors.
  local catalog_source_crd
  catalog_source_crd="$("${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null get customresourcedefinitions.apiextensions.k8s.io catalogsources.operators.coreos.com -o yaml)"
  local disable_security_context_config="# "
  local has_scc_key
  has_scc_key="$(yq eval '[.. | select(key == "grpcPodConfig")].[].properties | has("securityContextConfig")' - <<< "$catalog_source_crd")"
  if [[ "$has_scc_key" == "true" ]]; then
      disable_security_context_config=""
  fi

  env -i PATH="${PATH}" \
    NAMESPACE="${operator_ns}" \
    INDEX_IMAGE_REPO="${index_image_repo}" \
    INDEX_IMAGE_TAG="${index_image_tag}" \
    STARTING_CSV="rhacs-operator.${starting_csv_version}" \
    OPERATOR_CHANNEL="${operator_channel}" \
    DISABLE_SECURITY_CONTEXT_CONFIG="${disable_security_context_config}" \
    envsubst < "${ROOT_DIR}/operator/hack/operator.envsubst.yaml" \
    | "${ROOT_DIR}/operator/hack/retry-kubectl.sh" -n "${operator_ns}" apply -f -
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
  local -r csv_version="$1"
  local -r allow_dirty_tag="$2"

  if [[ "$csv_version" == *-dirty ]]; then
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
  local -r csv_version="$2"

  log "Waiting for an install plan to be created"
  if ! retry 15 5 "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null -n "${operator_ns}" wait subscription.operators.coreos.com stackrox-operator-test-subscription --for condition=InstallPlanPending --timeout=60s; then
    log "Install plan failed to materialize."
    gather_olm_resources "${operator_ns}"
    return 1
  fi

  log "Verifying that the subscription is progressing to the expected CSV of ${csv_version}..."
  local current_csv
  # `local` ignores `errexit` so we assign value separately: http://mywiki.wooledge.org/BashFAQ/105
  current_csv=$("${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null get -n "${operator_ns}" subscription.operators.coreos.com stackrox-operator-test-subscription -o jsonpath="{.status.currentCSV}")
  readonly current_csv
  local -r expected_csv="rhacs-operator.${csv_version}"
  if [[ $current_csv != "$expected_csv" ]]; then
    log "Subscription is progressing to unexpected CSV '${current_csv}', expected '${expected_csv}'"
    gather_olm_resources "${operator_ns}"
    return 1
  fi

  local install_plan_name
  # `local` ignores `errexit` so we assign value separately: http://mywiki.wooledge.org/BashFAQ/105
  install_plan_name=$("${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null get -n "${operator_ns}" subscription.operators.coreos.com stackrox-operator-test-subscription -o jsonpath="{.status.installPlanRef.name}")
  readonly install_plan_name

  log "Approving install plan ${install_plan_name}"
  retry 3 5 "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null -n "${operator_ns}" patch installplan "${install_plan_name}" --type merge -p '{"spec":{"approved":true}}'
}

function nurse_deployment_until_available() {
  local -r operator_ns="$1"
  local -r csv_version="$2"

  if ! wait_for_csv_success "${operator_ns}" "${csv_version}"; then
    log "CSV failed to enter Succeeded phase in time."
    gather_olm_resources "${operator_ns}"
    return 1
  fi

  # Double-check that the deployment itself is healthy.
  log "Making sure the ${csv_version} operator deployment is available..."
  if ! retry 3 5 "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null -n "${operator_ns}" wait deployments.apps -l "olm.owner=rhacs-operator.${csv_version}" --for condition=available --timeout 5s; then
    log "ACS Operator failed to become healthy in time after CSV finished installing."
    gather_olm_resources "${operator_ns}"
    return 1
  fi
}

function wait_for_csv_success() {
  local -r operator_ns="$1"
  local -r csv_version="$2"

  # We check the CSV status first, because it is hard to wait for the deployment in a non-racy way:
  # the deployment .status is set separately from the .spec, so the .status reflects the status of
  # the _old_ .spec until the deployment controller runs the first reconciliation.
  # We use kuttl because CSV has a Condition type incompatible with `kubectl wait`.
  log "Waiting for the ${csv_version} CSV to finish installing."
  "${KUTTL}" assert --timeout 600 --namespace "${operator_ns}" /dev/stdin <<-END
apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: rhacs-operator.${csv_version}
status:
  phase: Succeeded
END
}

function gather_olm_resources() {
  local -r operator_ns="$1"
  local -r msg="Gathering OLM resources for troubleshooting..."

  log "${msg}"
  log "Dumping install plans..."
  "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null -n "${operator_ns}" describe "installplan.operators.coreos.com" || true
  log "Dumping pod descriptions..."
  "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null -n "${operator_ns}" describe pods -l "olm.catalogSource=stackrox-operator-test-index" || true
  log "Dumping catalog sources and subscriptions..."
  "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null -n "${operator_ns}" describe "subscription.operators.coreos.com,catalogsource.operators.coreos.com" || true
  log "Dumping jobs..."
  "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null -n "${operator_ns}" describe "job.batch" || true
  if [[ -n ${CI:-} ]]; then
    local -r path="${ROX_CI_OUTPUT_DIR:-/tmp/k8s-service-logs}/olm-must-gather"
    log "Running oc adm must-gather in ${path} (which will be collected along with other CI artifacts)..."
    mkdir -p "${path}"
    ( cd "${path}" && run_oc_adm_must_gather >> "oc-adm-must-gather-output.txt"; )
  fi
  log "Resource collection completed, look before '${msg}' above for the cause of the failure."
}

function run_oc_adm_must_gather() {
  if ! oc adm must-gather 2>&1; then
    log "Running oc adm must-gather failed, perhaps this is not an OpenShift cluster?"
  fi
}

# OLMv1-specific functions

function get_ocp_version() {
  # Returns OCP version as X.Y.Z, or empty string if not an OpenShift cluster
  "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null get clusterversion -o jsonpath='{.items[0].status.desired.version}' 2>/dev/null || echo ""
}

function should_use_olmv1() {
  # Returns 0 (true) if OLMv1 should be used based on OCP version >= 4.20.1
  # Returns 1 (false) otherwise
  local ocp_version
  ocp_version="$(get_ocp_version)"

  if [[ -z "$ocp_version" ]]; then
    log "Not an OpenShift cluster, using OLMv0"
    return 1
  fi

  if [[ ! "$ocp_version" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
    log "Could not parse OCP version '$ocp_version', defaulting to OLMv0"
    return 1
  fi

  local major="${BASH_REMATCH[1]}"
  local minor="${BASH_REMATCH[2]}"
  local patch="${BASH_REMATCH[3]}"

  # Check if version >= 4.20.1
  if [[ $major -gt 4 ]] || \
     [[ $major -eq 4 && $minor -gt 20 ]] || \
     [[ $major -eq 4 && $minor -eq 20 && $patch -ge 1 ]]; then
    log "OCP version $ocp_version >= 4.20.1, using OLMv1"
    return 0
  fi

  log "OCP version $ocp_version < 4.20.1, using OLMv0"
  return 1
}

function wait_for_clusterextension_installed() {
  local -r operator_ns="$1"
  local -r extension_name="$2"
  local -r expected_version="$3"

  log "Waiting for ClusterExtension ${extension_name} to reach Installed status..."

  # Wait for up to 10 minutes for the extension to install
  # Catalog unpacking can take 60-90 seconds on first use
  local max_attempts=60
  local attempt=0
  while (( attempt < max_attempts )); do
    (( attempt++ ))

    local installed_status
    installed_status=$("${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null get clusterextension "${extension_name}" -o jsonpath='{.status.conditions[?(@.type=="Installed")].status}' 2>/dev/null || echo "")

    if [[ "$installed_status" == "True" ]]; then
      local installed_version
      installed_version=$("${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null get clusterextension "${extension_name}" -o jsonpath='{.status.install.bundle.version}' 2>/dev/null || echo "")

      if [[ "$installed_version" == "$expected_version" ]]; then
        log "ClusterExtension ${extension_name} successfully installed version ${installed_version}"
        return 0
      else
        log "ClusterExtension installed with version ${installed_version} (expected ${expected_version}), continuing to wait..."
        # Continue waiting - version might update as catalog unpacks
      fi
    fi

    # Check if it's in a failed state (but not Retrying, which is normal during catalog unpacking)
    local progressing_status
    local progressing_reason
    progressing_status=$("${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null get clusterextension "${extension_name}" -o jsonpath='{.status.conditions[?(@.type=="Progressing")].status}' 2>/dev/null || echo "")
    progressing_reason=$("${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null get clusterextension "${extension_name}" -o jsonpath='{.status.conditions[?(@.type=="Progressing")].reason}' 2>/dev/null || echo "")

    if [[ "$progressing_status" == "False" ]] && [[ "$progressing_reason" == "Failed" ]]; then
      log "ClusterExtension ${extension_name} installation failed"
      local error_msg
      error_msg=$("${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null get clusterextension "${extension_name}" -o jsonpath='{.status.conditions[?(@.type=="Progressing")].message}' 2>/dev/null || echo "")
      log "Error: ${error_msg}"
      gather_olmv1_resources "${operator_ns}" "${extension_name}"
      return 1
    fi

    if (( attempt % 6 == 0 )); then
      # Log progress every minute (6 attempts × 10 seconds)
      log "Still waiting for ClusterExtension to install (${attempt}/${max_attempts} attempts, status: Installed=${installed_status}, Progressing=${progressing_reason})..."
    fi

    sleep 10
  done

  log "ClusterExtension ${extension_name} failed to install within timeout"
  gather_olmv1_resources "${operator_ns}" "${extension_name}"
  return 1
}

function inject_test_env_vars() {
  local -r operator_ns="$1"
  local -r deployment_name="${2:-rhacs-operator-controller-manager}"

  log "Injecting test environment variables into operator deployment..."

  # Patch the deployment to add NO_PROXY and ROX_ADMISSION_CONTROLLER_CONFIG env vars
  # These are required for operator e2e test assertions
  "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null patch deployment "${deployment_name}" -n "${operator_ns}" --type='json' -p='[
    {"op": "add", "path": "/spec/template/spec/containers/0/env/-", "value": {"name": "NO_PROXY", "value": "127.1.2.3/8"}},
    {"op": "add", "path": "/spec/template/spec/containers/0/env/-", "value": {"name": "ROX_ADMISSION_CONTROLLER_CONFIG", "value": "true"}}
  ]'

  if [[ $? -eq 0 ]]; then
    log "Successfully injected test environment variables"
  else
    log "Warning: Failed to inject test environment variables, tests may fail"
  fi
}

function gather_olmv1_resources() {
  local -r operator_ns="$1"
  local -r extension_name="$2"
  local -r msg="Gathering OLMv1 resources for troubleshooting..."

  log "${msg}"
  log "Dumping ClusterExtension..."
  "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null describe clusterextension "${extension_name}" || true
  log "Dumping ClusterExtension status..."
  "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null get clusterextension "${extension_name}" -o yaml || true
  log "Dumping operator deployment..."
  "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null -n "${operator_ns}" describe deployments || true
  log "Dumping pod descriptions..."
  "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null -n "${operator_ns}" describe pods || true
  log "Dumping pod logs..."
  "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null -n "${operator_ns}" logs -l app=rhacs-operator --tail=100 || true

  if [[ -n ${CI:-} ]]; then
    local -r path="${ROX_CI_OUTPUT_DIR:-/tmp/k8s-service-logs}/olmv1-must-gather"
    log "Running oc adm must-gather in ${path} (which will be collected along with other CI artifacts)..."
    mkdir -p "${path}"
    ( cd "${path}" && run_oc_adm_must_gather >> "oc-adm-must-gather-output.txt"; )
  fi
  log "Resource collection completed, look before '${msg}' above for the cause of the failure."
}
