#!/usr/bin/env bash
set -eou pipefail

DEFAULT_KUBECONFIG="$HOME/.kube/config"
DEMO_LIFESPAN=72h

MENU_OPTIONS=()
ARTIFACTS_DIR="artifacts"

main() {
  local action="${1:-}"
  MENU_OPTIONS=(
    "Merge kubeconfigs for RC OpenShift cluster"
    "Merge kubeconfigs for qa GKE cluster"
    "Remove kubeconfigs for non-existing clusters"
    "Generate Slack message for RC OpenShift cluster"
    "Generate Slack message for qa GKE cluster"
    "Create new RC OpenShift cluster"
    "Create new qa GKE cluster"
    "Create new long-running cluster"
    "Quit"
  )
  if [[ -n "$action" ]]; then
    exec_option "$action"
    exit 0
  fi
  PS3="Choose action: "
  RED='\033[0;31m'
  NC='\033[0m' # No Color
  echo -e "${RED}WARNING:${NC} some of these scripts may be outdated, bleeding-edge, or not working. Read the code before you run them to be on the safe side."
  select ans in "${MENU_OPTIONS[@]}"
  do
    exec_option "$ans"
  done
}

exec_option() {
  local num_options="${#MENU_OPTIONS[@]}"
  local last_option="$((num_options-1))"
  case "$1" in
    "${MENU_OPTIONS[$last_option]}"|"$((last_option+1))"|q|Q) exit 0;;&
    "${MENU_OPTIONS[0]}"|1)
        cluster_name="$(get_cluster_name openshift)"
        merge_kubeconfigs "$cluster_name" "$DEFAULT_KUBECONFIG"
        exit 0
        ;;
    "${MENU_OPTIONS[1]}"|2)
        cluster_name="$(get_cluster_name gke)"
        merge_kubeconfigs "$cluster_name" "$DEFAULT_KUBECONFIG"
        exit 0
        ;;
    "${MENU_OPTIONS[2]}"|3)
        cleanup_artifacts  "$DEFAULT_KUBECONFIG"
        exit 0
        ;;
    "${MENU_OPTIONS[3]}"|4)
        generate_slack_message_for_openshift
        exit 0
        ;;
    "${MENU_OPTIONS[4]}"|5)
        generate_slack_message_for_gke
        exit 0
        ;;
    "${MENU_OPTIONS[5]}"|6)
        create_rc_openshift_cluster
        exit 0
        ;;
    "${MENU_OPTIONS[6]}"|7)
        create_qa_gke_cluster
        exit 0
        ;;
    "${MENU_OPTIONS[7]}"|8)
        create_long_running_cluster
        exit 0
        ;;
    *) echo "invalid option: '$1'";;
  esac
}

cluster_ready() {
  local cluster_name="$1"
  infractl get "$cluster_name" | grep -xq "Status:      READY"
}

check_cluster_status() {
  local cluster_name="$1"
  status="$( infractl get "$cluster_name" | awk '{if ($1 == "Status:") print $2}' )"

  echo "$status"
}

does_cluster_exist() {
  local cluster_name="$1"
  nline="$({ infractl get "$cluster_name" 2> /dev/null || true; } | wc -l)"
  if (("$nline" == 0)); then
    return 1
  else
    status="$(check_cluster_status "$cluster_name")"
    if [[ "$status" == "FINISHED" ]]; then
      return 1
    else
      return 0
    fi
  fi
}

wait_for_cluster_to_be_ready() {
  local cluster_name="$1"

  while ! cluster_ready "${cluster_name}"; do
    echo "Cluster ${cluster_name} not ready yet. Waiting 15 seconds..."
    sleep 15
  done
}

ensure_cluster_exists() {
  local cluster_name="$1"

  infractl get "${cluster_name}" || die "cluster '${cluster_name}' not found"
}

fetch_artifacts() {
  local cluster_name="$1"

  infractl artifacts "${cluster_name}" --download-dir "${ARTIFACTS_DIR}/${cluster_name}" > /dev/null 2>&1
  while ! test -d "${ARTIFACTS_DIR}/${cluster_name}"; do
    echo "Waiting until artifacts download"
    sleep 5
  done
}

get_cluster_postfix() {
  echo "${RELEASE//./-}-rc${RC_NUMBER}-test" # Change before merging
}

get_cluster_prefix() {
  local cluster_type="$1"

  if [[ "$cluster_type" == "openshift" ]]; then
    echo "os4-9-demo"
  elif [[ "$cluster_type" == "gke" ]]; then
    echo "qa-demo"
  elif [[ "$cluster_type" == "long-running" ]]; then
    echo "$cluster_type"
  else
    die "Unknown cluster type: $cluster_type"
  fi
}

get_cluster_name() {
  local cluster_type="$1"

  cluster_prefix="$(get_cluster_prefix "$cluster_type")"
  cluster_postfix="$(get_cluster_postfix)"
  cluster_name="${cluster_prefix}-${cluster_postfix}"

  echo "$cluster_name"
}

create_rc_openshift_cluster() {
  require_binary infractl
  require_binary oc

  [[ -n "${INFRA_TOKEN}" ]] || die "INFRA_TOKEN is not set"
  [[ -n "$RC_NUMBER" ]] || die "RC_NUMBER undefined"
  [[ -n "$RELEASE" ]] || die "RELEASE undefined"

  CLUSTER_NAME="$(get_cluster_name openshift)"
  export CLUSTER_NAME
  infractl create openshift-4-demo "${CLUSTER_NAME}" --lifespan "$DEMO_LIFESPAN" --arg openshift-version=ocp/stable-4.9 || echo "Cluster creation already started or the cluster already exists"

  wait_for_cluster_to_be_ready "$CLUSTER_NAME"
  ensure_cluster_exists "$CLUSTER_NAME"
  fetch_artifacts "$CLUSTER_NAME"

  merge_kubeconfigs "$CLUSTER_NAME" "$DEFAULT_KUBECONFIG"
  export KUBECONFIG="$DEFAULT_KUBECONFIG"
  kubectl config use-context "ctx-${CLUSTER_NAME}" || die "cannot switch kubectl context to ctx-${CLUSTER_NAME}"

  . "${ARTIFACTS_DIR}/${CLUSTER_NAME}/dotenv"
  oc login --username="$OPENSHIFT_CONSOLE_USERNAME" --password="$OPENSHIFT_CONSOLE_PASSWORD"

  export MAIN_TAG="${RELEASE}.0-rc.${RC_NUMBER}"
  oc -n stackrox set image deploy/central "central=docker.io/stackrox/main:${MAIN_TAG}"
  oc -n stackrox patch hpa/scanner -p '{"spec":{"minReplicas":2}}'
  oc -n stackrox set image deploy/scanner "scanner=docker.io/stackrox/scanner:${MAIN_TAG}"
  oc -n stackrox set image deploy/scanner-db "db=docker.io/stackrox/scanner-db:${MAIN_TAG}"
  oc -n stackrox set image deploy/scanner-db "init-db=docker.io/stackrox/scanner-db:${MAIN_TAG}"
  oc -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","env":[{"name":"POD_NAMESPACE","valueFrom":{"fieldRef":{"fieldPath":"metadata.namespace"}}}],"volumeMounts":[{"name":"cache","mountPath":"/var/cache/stackrox"}]}],"volumes":[{"name":"cache","emptyDir":{}}]}}}}'
  oc -n stackrox set image deploy/sensor "sensor=docker.io/stackrox/main:${MAIN_TAG}"
  oc -n stackrox set image ds/collector "compliance=docker.io/stackrox/main:${MAIN_TAG}"
  oc -n stackrox set image ds/collector "collector=docker.io/stackrox/collector:${MAIN_TAG}"
  oc -n stackrox set image deploy/admission-control "admission-control=docker.io/stackrox/main:${MAIN_TAG}"

  oc -n stackrox get deploy,pods -o wide
}

create_qa_gke_cluster() {
  require_binary infractl

  [[ -n "${INFRA_TOKEN}" ]] || die "INFRA_TOKEN is not set"

  CLUSTER_NAME="$(get_cluster_name gke)"
  export CLUSTER_NAME

  if does_cluster_exist "$CLUSTER_NAME"; then
    echo "Cluster $CLUSTER_NAME already exists"
  else
    infractl create qa-demo "${CLUSTER_NAME}" --arg "main-image=docker.io/stackrox/main:${RELEASE}.${PATCH_NUMBER}-rc.${RC_NUMBER}" --lifespan "$DEMO_LIFESPAN"
  fi

  wait_for_cluster_to_be_ready "$CLUSTER_NAME"
  ensure_cluster_exists "$CLUSTER_NAME"
  fetch_artifacts "$CLUSTER_NAME"

  export KUBECONFIG="${ARTIFACTS_DIR}/${CLUSTER_NAME}/kubeconfig"

  kubectl -n stackrox get pods
}

create_long_running_cluster() {

  CLUSTER_NAME="$(get_cluster_name long-running)"
  export CLUSTER_NAME

  echo "cluster_name= $CLUSTER_NAME"
  
  status="$(check_cluster_status "$CLUSTER_NAME")"
  
  if does_cluster_exist "$CLUSTER_NAME"; then
      echo "Unable to create cluster"
  else
      infractl create gke-default $CLUSTER_NAME --lifespan 168h --arg nodes=5 --wait --slack-me
  fi
  
  # Set your local kubectl context to the remote cluster once the above completes successfully.
  infractl get $CLUSTER_NAME --json | jq '.Connect' -r | bash
  
  
  export MAIN_IMAGE_TAG=$(git describe --tags --abbrev=0) # Release version, e.g. 3.63.0-rc.2.
  export API_ENDPOINT="localhost:8000"
  
  export STORAGE=pvc # Backing storage
  export STORAGE_CLASS=faster # Runs on an SSD type
  export STORAGE_SIZE=100 # 100G
  export MONITORING_SUPPORT=true # Runs monitoring
  export LOAD_BALANCER=lb
  
  toplevel_dir="$(git rev-parse --show-toplevel)"
  "$toplevel_dir/deploy/k8s/central.sh" # Launches central
  
  # Open port-forward to central, e.g. with
  kubectl -n stackrox port-forward deploy/central 8000:8443 > /dev/null 2>&1 &
  sleep 60
  
  export ROX_ADMIN_USERNAME=admin
  
  export ROX_ADMIN_PASSWORD="$(cat deploy/k8s/central-deploy/password)"

  "$toplevel_dir/deploy/k8s/sensor.sh"

  kubectl -n stackrox set env deploy/sensor MUTEX_WATCHDOG_TIMEOUT_SECS=0
  kubectl -n stackrox set env deploy/sensor ROX_FAKE_KUBERNETES_WORKLOAD=long-running
  kubectl -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"requests":{"memory":"3Gi","cpu":"2"},"limits":{"memory":"12Gi","cpu":"4"}}}]}}}}'
  
  kubectl -n stackrox set env deploy/central MUTEX_WATCHDOG_TIMEOUT_SECS=0

  "$toplevel_dir/scale/launch_workload.sh np-load"
}

cleanup_artifacts() {
  local kubeconfig_location="${1:-"$DEFAULT_KUBECONFIG"}"
  local dead_clusters=()
  readarray -d '' ALL_ARTIFACTS < <(find ${ARTIFACTS_DIR} -maxdepth 1 -mindepth 1 -type d -print0)
  for path in "${ALL_ARTIFACTS[@]}"; do
    local cluster_name
    cluster_name="$(basename "$path")"
    echo "Checking cluster '$cluster_name'"
    if ! infractl artifacts "$cluster_name" -d "${ARTIFACTS_DIR}/${cluster_name}" > /dev/null 2>&1 ; then
      echo "Cluster '$cluster_name' is gone - removing artifacts"
      dead_clusters+=("$cluster_name")
    else
      echo "Cluster '$cluster_name' is still running"
    fi
  done
  rm -rf "${dead_clusters[@]}"
}

merge_kubeconfigs() {
  local cluster_name="$1"

  local kubeconfig_location="${2:-"$DEFAULT_KUBECONFIG"}"
    # remove ':' from prefix and suffix
  kubeconfig_location="${kubeconfig_location#:}"
  kubeconfig_location="${kubeconfig_location%:}"
  if [[ "$kubeconfig_location" == *:* ]]; then
    echo "Kubeconfig location is a concatenation of multiple paths. Will use standard '$HOME/.kube/config' instead"
    kubeconfig_location="$HOME/.kube/config"
  fi

  require_binary readlink || require_binary greadlink
  local readlink_bin
  # We want to use GNU readlink, so on Mac this is 'greadlink', while on Linux this is 'readlink'
  readlink_bin="$(command -v greadlink)"
  if [[ -z "$readlink_bin" ]]; then
    readlink_bin="$(command -v readlink)"
  fi

  [[ -n "$RC_NUMBER" ]] || die "RC_NUMBER undefined"
  [[ -n "$RELEASE" ]] || die "RELEASE undefined"
  DIR="${ARTIFACTS_DIR}/${cluster_name}"
  [[ -d "$DIR" ]] || die "DIR not found: '$DIR'"

  # KUBECONFIGS_STR contains list of paths (concatenated with ':') to kubeconfig files
  KUBECONFIGS_STR=""

  # rename context to ctx-$clustername
  readarray -d '' ARTIFACTS_KUBE < <(find ${ARTIFACTS_DIR} -type f -name kubeconfig -print0)
  for kube_rel_path in "${ARTIFACTS_KUBE[@]}"; do
    cluster="$(basename "$(dirname "$kube_rel_path")")"
    if ! infractl artifacts "$cluster" -d "${ARTIFACTS_DIR}/${cluster}" > /dev/null 2>&1 ; then
      echo "Skipping cluster: '$cluster' - cannot download artifacts"
      continue
    fi

    kube_path="$("$readlink_bin" -f "${kube_rel_path}")"
    current_context_name="$(kubectl --kubeconfig "${kube_path}" config get-contexts -o name)"
    echo "Found context '${current_context_name}' in '${kube_path}'"
    if [[ -z "$current_context_name" ]]; then
      echo "Skipping cluster: '$cluster' - unable to find any context"
      continue
    fi
    if [[ "$current_context_name" != "ctx-${cluster}" ]]; then
      kubectl --kubeconfig "${kube_path}" config rename-context "${current_context_name}" "ctx-${cluster}"
    fi

    # rename user to guarantee unique names (FIXME: works only for admin)
    # TODO(RS-433): Remove after adressing in the infra
    gsed -i "s/user: admin/user: admin-${cluster}/g" "${kube_path}"
    gsed -i "s/^- name: admin$/- name: admin-${cluster}/g" "${kube_path}"

    if [[ -f "$kube_path" ]]; then
      KUBECONFIGS_STR="${KUBECONFIGS_STR}:${kube_path}"
    fi
  done

  echo "KUBECONFIGS_STR='$KUBECONFIGS_STR'"
  # remove ':' from prefix and suffix
  KUBECONFIGS_STR="${KUBECONFIGS_STR#:}"
  KUBECONFIGS_STR="${KUBECONFIGS_STR%:}"
  # backup default kubeconfig, but do not overwrite the backup if it already exists
  if [[ ! -f "${kubeconfig_location}.bak" ]]; then
    cp "$kubeconfig_location" "${kubeconfig_location}.bak"
  fi
  #  merge kubeconfigs into the default kubeconfig
  KUBECONFIG="$KUBECONFIGS_STR" kubectl config view --raw > "$kubeconfig_location"

  if [[ "$KUBECONFIG" != "$kubeconfig_location" ]]; then
    echo "Non-standard KUBECONFIG location '$KUBECONFIG'"
    echo "You may want to run: export KUBECONFIG=\"$kubeconfig_location\""
  fi
  # confirm visually
  kubectl config get-contexts
}

generate_slack_message_for_openshift() {
  local cluster_name
  cluster_name="$(get_cluster_name openshift)"
  [[ -n "$RC_NUMBER" ]] || die "RC_NUMBER undefined"
  [[ -n "$RELEASE" ]] || die "RELEASE undefined"

  DIR="${ARTIFACTS_DIR}/$cluster_name"
  [[ -d "$DIR" ]] || die "DIR not found: '$DIR'"

  . "${DIR}/dotenv"

  [[ -n "$OPENSHIFT_CONSOLE_USERNAME" ]] || die "OPENSHIFT_CONSOLE_USERNAME undefined"
  [[ -n "$OPENSHIFT_CONSOLE_PASSWORD" ]] || die "OPENSHIFT_CONSOLE_PASSWORD undefined"
  [[ -n "$OPENSHIFT_VERSION" ]] || die "OPENSHIFT_VERSION undefined"
  [[ -f "${DIR}/url-openshift" ]] || die "url-openshift file not found in ${DIR}"
  [[ -f "${DIR}/url-stackrox" ]] || die "url-stackrox file not found in ${DIR}"
  [[ -f "${DIR}/admin-password" ]] || die "admin-password file not found in ${DIR}"

  cat <<-EOF
:openshift: Openshift \`${OPENSHIFT_VERSION}\` cluster with \`${RELEASE}-rc${RC_NUMBER}\`

:computer: Console: $(cat "${DIR}/url-openshift")
Username: \`${OPENSHIFT_CONSOLE_USERNAME}\`
Password: \`${OPENSHIFT_CONSOLE_PASSWORD}\`

:computer: Central: $(cat "${DIR}/url-stackrox")
Username: \`admin\`
Password: \`$(cat "${DIR}/admin-password")\`
EOF
}

generate_slack_message_for_gke() {
  local cluster_name
  cluster_name="$(get_cluster_name gke)"
  [[ -n "$RC_NUMBER" ]] || die "RC_NUMBER undefined"
  [[ -n "$RELEASE" ]] || die "RELEASE undefined"

  DIR="${ARTIFACTS_DIR}/$cluster_name"
  [[ -d "$DIR" ]] || die "DIR not found: '$DIR'"

  [[ -f "${DIR}/url" ]] || die "url file not found in ${DIR}"

  cat <<-EOF
:qke: GKE cluster with \`${RELEASE}-rc${RC_NUMBER}\`

url: \`$(cat "${DIR}/url")\`
EOF
}

die() {
  >&2 echo "$@"
  exit 1
}

require_binary() {
  command -v "${1}" > /dev/null || die "Install ${1}"
}

main "$@"
