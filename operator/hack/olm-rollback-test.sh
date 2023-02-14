#!/usr/bin/bash

set -e

source "$(dirname "$0")/common.sh"

function install_olm() {
  if [ $(kubectl get deploy -n olm -o json | jq '.items | length') == 3 ]; then
      log OLM is already installed
  else
      log Installing OLM
      # Download operator-sdk binary here:
      # https://github.com/operator-framework/operator-sdk/releases/tag/v1.25.4
      operator-sdk olm install
  fi
}

function init() {
  install_olm

  ns_list=$(kubectl get ns --no-headers)

  log "Installing pull secrets in namespaces"
  create_pull_secret "${operator_ns}" "quay.io"

  if [ "$(echo \"$ns_list\" | grep ${central_ns})" == "" ]; then
    kubectl create ns ${central_ns}
  fi

  create_pull_secret "${central_ns}" "quay.io"
}

function install_cert_manager() {
  log "Installing cert manager"
  cat <<EOF | kubectl apply -n ${operator_ns} -f -
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: cert-manager
spec:
  channel: stable
  name: cert-manager
  source: operatorhubio-catalog
  sourceNamespace: olm
  installPlanApproval: Automatic
EOF
}

function install_acs_catalog_source {
  version=$1
  log "Installing ACS CatalogSource at version ${version}"
  cat <<EOF | kubectl apply -n ${operator_ns} -f -
---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: rhacs
spec:
  displayName: Advanced Cluster Security
  grpcPodConfig:
    securityContextConfig: restricted
  image: ${image_spec_prefix}:v${version}
  publisher: Red Hat
  secrets:
  - operator-pull-secret
  sourceType: grpc
  updateStrategy:
    registryPoll:
      interval: 60m
EOF
}

function install_acs_operator {
  log "Installing ACS operator"
  version=$1
  cat <<EOF | kubectl apply -n ${operator_ns} -f -
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: rhacs
spec:
  channel: latest
  name: rhacs-operator
  source: rhacs
  sourceNamespace: ${operator_ns}
  installPlanApproval: Automatic  # Choosing automatic to emulate how most customers configure subscriptions (afiact)
  config:
    env:
    # use a test value for NO_PROXY. This will not have any impact
    # on the services at runtime, but we can test if it gets piped
    # through correctly.
    - name: NO_PROXY
      value: "127.1.2.3/8"
EOF

  nurse_deployment_until_available "${operator_ns}" "${version}"
}

function install_central() {
  # NOTE: Use helm or some templating tool if this gets any more complicated
  log "Installing Central CR"
  if [[ "$force_postgres" == true ]]; then
    read -d '' central_spec << EOF || true
  central:
    db:
      isEnabled: Enabled
    adminPasswordSecret:
      name: admin-pass
EOF
  else
    read -d '' central_spec << EOF || true
  central:
    adminPasswordSecret:
      name: admin-pass
EOF
  fi

  cat <<EOF | kubectl apply -n ${central_ns} -f -
---
apiVersion: platform.stackrox.io/v1alpha1
kind: Central
metadata:
  name: stackrox-central-services
spec:
  imagePullSecrets:
  - name: operator-pull-secret
  ${central_spec}
---
apiVersion: v1
kind: Secret
metadata:
  name: admin-pass
data:
  # letmein
  password: bGV0bWVpbg==
EOF
}

function test_central_api() {
  log "Waiting for Central to be ready"
  kubectl -n ${central_ns} wait --for=condition=Deployed --timeout=5m central/stackrox-central-services
  kubectl -n ${central_ns} wait --for=condition=Available --timeout=5m deploy/central

  log "Valiating that Central API is available"
  kubectl -n ${central_ns} port-forward svc/central 44444:443 &
  sleep 2 # allow for port forward to get set up
  test_result=$(curl -k -u 'admin:letmein' https://localhost:44444/api/docs/swagger -s | jq -r .info.title)
  kill %1

  if [ "${test_result}" != "API Reference" ]; then
      log Failed to query API
      exit 1
  fi
}

function update_acs_catalog_source() {
  version=$1

  log "Updating ACS CatalogSource to ${version}"
  kubectl -n ${operator_ns} patch catalogsource/rhacs --type merge -p '{"spec":{"image":"'"${image_spec_prefix}"':v'"${version}"'"}}'

  sleep 5

  log "Waiting for CatalogSource to be ready"
  new_pod=$(kubectl get pod -n ${operator_ns} -l olm.catalogSource=rhacs -o json | \
    jq -r '.items[] | select(.metadata.deletionTimestamp == null) | .metadata.name')

  kubectl -n ${operator_ns} wait \
    --timeout=300s \
    --for=condition=Ready \
    pod/${new_pod}

  "${KUTTL}" assert --timeout 300 --namespace ${operator_ns} /dev/stdin <<-END
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: rhacs
status:
  connectionState:
    lastObservedState: READY
END
}

function get_previous_db_version() {
  log "Querying Central pod for previous DB version"
  pod_name=$(kubectl -n acs get pod -l app=central -o name)
  prev_db_v=$(kubectl -n ${central_ns} exec $pod_name -- \
     cat /var/lib/stackrox/.previous/migration_version.yaml | \
     grep '^image:' | \
     cut -f 2 -d : | \
     tr -d ' ')
  echo $prev_db_v
}

function patch_db_rollback_version() {
  version=$1
  log "Patching central-config: forceRollbackVersion: ${version}"

  kubectl -n ${central_ns} get configmap central-config -o yaml | \
    sed -e "s/forceRollbackVersion: none/forceRollbackVersion: ${version}/" | \
    kubectl -n ${central_ns} apply -f -

  kubectl -n ${central_ns} delete $(kubectl -n ${central_ns} get pod -l app=central -o name)
}

function remove_acs_operator() {
  version=$1
  log "Removing ACS CSV and Subscription"
  kubectl -n ${operator_ns} delete csv rhacs-operator.v${version}
  kubectl -n ${operator_ns} delete subscription rhacs
}

function ensure_central_at_version() {
  version=$1
  log "Waiting for Central to be upgraded to version ${version}"
  "${KUTTL}" assert --timeout 300 --namespace ${central_ns} /dev/stdin <<-END
apiVersion: platform.stackrox.io/v1alpha1
kind: Central
metadata:
  name: stackrox-central-services
status:
  productVersion:  $(echo ${version} | sed s/\.0-/.x-/)
END
}

function csv_version_tag() {
  echo $(kubectl -n "${operator_ns}" get csv  -l operators.coreos.com/rhacs-operator.operators -o json | jq -r '.items[].spec.version' | sort | tail -n 1)
}

function nurse_olm_upgrade() {
  initial_version_tag="$1"
  desired_version_tag="$2"

  local current_version_tag="$(csv_version_tag)"

  # Wait until a new CSV has been created
  while [[ "${current_version_tag}" == "${initial_version_tag}" ]]; do
      sleep 1
      current_version_tag="$(csv_version_tag)"
  done

  # Nurse the deployment for each intermediate CSV
  while [[ "${current_version_tag}" != "${desired_version_tag}" ]]; do
      echo Current: ${current_version_tag}
      echo Desired: ${desired_version_tag}
      nurse_deployment_until_available ${operator_ns} ${current_version_tag}
      while [[ "$(csv_version_tag)" == "${current_version_tag}" ]]; do
          # Wait until the new CSV is created
          sleep 10
      done
      current_version_tag="$(csv_version_tag)"
  done

  # Nurse the deployment for the final CSV
  nurse_deployment_until_available ${operator_ns} ${desired_version_tag}
}

function main() {
  POSITIONAL_ARGS=()
  force_postgres=false
  openshift=false
  image_spec_prefix=quay.io/rhacs-eng/stackrox-operator-index
  operator_ns=operators
  central_ns=acs

  while [[ $# -gt 0 ]]; do
    case $1 in
      -p|--force-postgres)
        force_postgres=true
        shift
        ;;
      -o|--openshift)
        openshift=true
        shift
        ;;
      -i|--image-spec-prefix)
        image_spec_prefix="$2"
        shift
        shift
        ;;
      -*|--*)
        echo "Unknown option $1"
        exit 1
        ;;
      *)
        POSITIONAL_ARGS+=("$1") # save positional arg
        shift # past argument
        ;;
    esac
  done

  set -- "${POSITIONAL_ARGS[@]}" # restore positional parameters

  old_version="${1:-}"
  new_version="${2:-}"

  if [[ "$old_version" == "" || "$new_version" == "" ]]; then
      echo "Usage:   $0 [-poi] <old_version> <new_version>"
      echo "Example: $0 3.73.0 3.74.0-91-gacfe66b6fa"
      exit 1
  fi

  if [[ "$openshift" == true ]]; then
    echo "Openshift option is not available yet"
    exit 1
  fi

  init
  install_cert_manager
  install_acs_catalog_source ${old_version}
  install_acs_operator ${old_version}
  install_central
  test_central_api
  update_acs_catalog_source ${new_version}
  nurse_olm_upgrade "${old_version}" "${new_version}"
  test_central_api

  # rollback
  previous_db_version="$(get_previous_db_version)"
  log "Previous version: ${previous_db_version}"
  remove_acs_operator "${new_version}"
  update_acs_catalog_source "${previous_db_version}"
  install_acs_operator "${previous_db_version}"
  ensure_central_at_version "${previous_db_version}"
  remove_acs_operator "${previous_db_version}"
  patch_db_rollback_version "${previous_db_version}"
  test_central_api
  install_acs_operator "${previous_db_version}"
  test_central_api
}

main "$@"
