#!/usr/bin/env bash

DEFAULT_KUBECONFIG="$HOME/.kube/config"

MENU_OPTIONS=(
  "Merge kubeconfigs"
  "Generate Slack message for cluster '${cluster_name}'"
  "Create new RC OpenShift cluster '${cluster_name}'"
  "Quit"
)

main() {
  local action="${1}"
  local cluster_prefix="${2:-os4-9-demo}"
  local cluster_name="${cluster_prefix}-${RELEASE//./-}-rc${RC_NUMBER}"
  PS3="Choose action: "
  if [[ -n "$action" ]]; then
    exec_option "$action"
    exit 0
  fi
  RED='\033[0;31m'
  NC='\033[0m' # No Color
  echo -e "${RED}WARNING:${NC} some of these scripts may be outdated, bleeding-edge, or not working. Read the code before you run them to be on the safe side."
  select ans in "${MENU_OPTIONS[@]}"
  do
    exec_option "$ans" "$cluster_prefix"
  done
}

exec_option() {
  local num_options="${#MENU_OPTIONS[@]}"
  local cluster_prefix="$2"
  local last_option="$((num_options-1))"
  case "$1" in
    "${MENU_OPTIONS[$last_option]}"|"$((last_option+1))"|q|Q) exit 0;;&
    "${MENU_OPTIONS[0]}"|1)
        [[ -n "$cluster_prefix" ]] || die "cluster_prefix required"
        merge_kubeconfigs "$cluster_prefix" "$DEFAULT_KUBECONFIG"
        exit 0
        ;;
    "${MENU_OPTIONS[1]}"|2)
        [[ -n "$cluster_prefix" ]] || die "cluster_prefix required"
        generate_slack_message "$cluster_prefix"
        exit 0
        ;;
    "${MENU_OPTIONS[2]}"|3)
        [[ -n "$cluster_prefix" ]] || die "cluster_prefix required"
        create_rc_openshift_cluster "$cluster_prefix"
        exit 0
        ;;
    *) echo "invalid option: '$1'";;
  esac
}

cluster_ready() {
  local cluster_name="$1"
  infractl get "$cluster_name" | grep -xq "Status:      READY"
}

create_rc_openshift_cluster() {
  require_binary infractl
  require_binary oc

  local cluster_prefix="$1"
  export CLUSTER_NAME="${cluster_prefix}-${RELEASE//./-}-rc${RC_NUMBER}"
  infractl create openshift-4-demo "${CLUSTER_NAME}" --lifespan 168h --arg openshift-version=ocp/stable-4.9 || echo "Cluster creation already started"
  ## wait
  while ! cluster_ready "${CLUSTER_NAME}"; do
    echo "Cluster ${CLUSTER_NAME} not ready yet. Waiting 15 seconds..."
    sleep 15
  done

  # ensure cluster exists
  infractl get "${CLUSTER_NAME}" || die "cluster '${CLUSTER_NAME}' not found"
  # fetch artifacts
  infractl artifacts "${CLUSTER_NAME}" -d "artifacts/${CLUSTER_NAME}" > /dev/null 2>&1

  while ! test -d "artifacts/${CLUSTER_NAME}"; do
    echo "Wainting until artifacts download"
    sleep 5
  done

  merge_kubeconfigs "$cluster_prefix" "$DEFAULT_KUBECONFIG"
  export KUBECONFIG="$DEFAULT_KUBECONFIG"
  kubectl config use-context "ctx-${CLUSTER_NAME}" || die "cannot switch kubectl context to ctx-${CLUSTER_NAME}"

  . "artifacts/${CLUSTER_NAME}/dotenv"
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

merge_kubeconfigs() {
  local DIR_PREFIX="${1:-os4-9-demo}"
  [[ -n "$DIR_PREFIX" ]] || die "DIR_PREFIX missing. Usage: merge_kubeconfigs <DIR_PREFIX> <KUBECONFIG_PATH>"

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
  CLUSTER_NAME="${DIR_PREFIX}-${RELEASE//./-}-rc${RC_NUMBER}"
  DIR="artifacts/${CLUSTER_NAME}"
  [[ -d "$DIR" ]] || die "DIR not found: '$DIR'"

  # KUBECONFIGS_STR contains list of paths (concatenated with ':') to kubeconfig files
  KUBECONFIGS_STR=""

  # rename context to ctx-$clustername
  readarray -d '' ARTIFACTS_KUBE < <(find artifacts -type f -name kubeconfig -print0)
  for kube_rel_path in "${ARTIFACTS_KUBE[@]}"; do
    cluster="$(basename "$(dirname "$kube_rel_path")")"
    if ! infractl artifacts "$cluster" -d "artifacts/${cluster}" > /dev/null 2>&1 ; then
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

generate_slack_message() {
  local DIR_PREFIX="${1:-os4-9-demo}"
  [[ -n "$RC_NUMBER" ]] || die "RC_NUMBER undefined"
  [[ -n "$RELEASE" ]] || die "RELEASE undefined"

  DIR="artifacts/$DIR_PREFIX-${RELEASE//./-}-rc${RC_NUMBER}"
  [[ -d "$DIR" ]] || die "DIR not found: '$DIR'"

  . "${DIR}/dotenv"

  [[ -n "$CLUSTER_NAME" ]] || die "CLUSTER_NAME undefined"
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

die() {
  >&2 echo "$@"
  exit 1
}

require_binary() {
  command -v "${1}" > /dev/null || die "Install ${1}"
}

main "$@"
