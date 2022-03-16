#!/bin/bash
set -eu
source "scripts/common.sh"
source "scripts/config.sh"

function docker_login {
  docker login docker.io
  docker login stackrox.io
  docker login collector.stackrox.io

  # docker login -u "$DOCKER_IO_PULL_USERNAME" -p "$DOCKER_IO_PULL_PASSWORD" docker.io
  # docker login -u "$STACKROX_IO_USERNAME"    -p "$STACKROX_IO_PASSWORD"    stackrox.io
  # docker login -u "$STACKROX_IO_USERNAME"    -p "$STACKROX_IO_PASSWORD"    collector.stackrox.io
}

function stackrox_teardown {
  cd "$STACKROX_SOURCE_ROOT"
  assert_file_exists "$STACKROX_TEARDOWN_SCRIPT"
  "$STACKROX_TEARDOWN_SCRIPT" <<<"yes"

  # Remove existing stackrox resources
  RESOURCE_KINDS=(cm deploy ds networkpolicy secret svc serviceaccount pv pvc
    clusterrole role validatingwebhookconfiguration clusterrolebinding psp
    rolebinding SecurityContextConstraints)
  RESOURCE_KINDS_STR=$(join_by "," "${RESOURCE_KINDS[@]}")
  kubectl -n stackrox delete "$RESOURCE_KINDS_STR" -l "app.kubernetes.io/name=stackrox" --wait || true
  kubectl delete -R -f scripts/ci/psp --wait || true
  kubectl delete ns stackrox --wait || true
  helm uninstall monitoring || true
  helm uninstall central || true
  helm uninstall scanner || true
  helm uninstall sensor || true
  if kubectl get namespace -o name | grep -qE '^namespace/qa'; then
     kubectl delete --wait namespace qa
  fi
}

# https://help-internal.stackrox.com/docs/get-started/quick-start/
function stackrox_deploy_via_helm {
  cd "$STACKROX_SOURCE_ROOT"
  helm plugin update diff >/dev/null  # https://github.com/databus23/helm-diff

  if cluster_is_openshift; then
    ./deploy/openshift/deploy.sh  # requires docker.io login `pass docker.io`
  else
    ./deploy/k8s/deploy.sh
  fi
}

# operates against current kube context
function port-forward-central {
  cd "$STACKROX_SOURCE_ROOT"
  pkill -f 'port-forward.*stackrox.*svc/central' || true
  sleep 3
  nohup kubectl port-forward -n stackrox svc/central 8443:443 &> /tmp/central.log &
  sleep 5
  pgrep -fl 'port-forward.*stackrox.*svc/central' || {
    warning "Port forwarding to central has failed"
    cat /tmp/central.log
  }

  # The Groovy e2e api tests require these two variables are set
  export API_HOSTNAME="localhost"
  export API_PORT="8443"

  nc -vz "$API_HOSTNAME" "$API_PORT" \
    || error "FAILED: [nc -vz $API_HOSTNAME $API_PORT]"
}


# __MAIN__
CURRENT_KUBE_CONTEXT=$(kubectl config current-context)
REQUIRED_KUBE_CONTEXT="default/api-sb-03-15-osdgcp-fu0z-s2-devshift-org:6443/admin"
[[ "$CURRENT_KUBE_CONTEXT" == "$REQUIRED_KUBE_CONTEXT" ]] \
  || error "kubecontext mismatch [$CURRENT_KUBE_CONTEXT] [$REQUIRED_KUBE_CONTEXT]"
export LOAD_BALANCER="lb"
export MONITORING_SUPPORT=true

docker_login
stackrox_teardown
kubectl delete --wait namespace qa &>/dev/null || true
kubectl create namespace qa
stackrox_deploy_via_helm
port-forward-central
kubectl delete -f "qa-tests-backend-single-step/scripts/scc-qatest-anyuid.yaml" || true
kubectl apply -f "qa-tests-backend-single-step/scripts/scc-qatest-anyuid.yaml"
echo "Cluster is ready for testing."
