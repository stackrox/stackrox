#!/bin/bash
set -eu
source "local-test-example/common.sh"
source "local-test-example/config.sh"

function docker_login {
  docker login docker.io
  docker login stackrox.io
  docker login collector.stackrox.io
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


# __MAIN__
export KUBECONFIG=/tmp/kubeconfig
CURRENT_KUBE_CONTEXT=$(kubectl config current-context)
[[ "$CURRENT_KUBE_CONTEXT" =~ default/api-.*-devshift-org:6443/admin ]] \
  || error "Unexpected kube econtext [$CURRENT_KUBE_CONTEXT]"
export LOAD_BALANCER="lb"
export MONITORING_SUPPORT=true
REGISTRY_USERNAME=$(pass docker.io | jq -r '.username')
REGISTRY_PASSWORD=$(pass docker.io | jq -r '.password')
export REGISTRY_USERNAME REGISTRY_PASSWORD

docker_login
stackrox_teardown
kubectl delete --wait namespace qa &>/dev/null || true
kubectl create namespace qa
stackrox_deploy_via_helm

cd "$STACKROX_SOURCE_ROOT/qa-tests-backend"
kubectl delete -f "src/k8s/scc-qatest-anyuid.yaml" &>/dev/null || true
kubectl apply -f "src/k8s/scc-qatest-anyuid.yaml"
echo "Cluster is ready for testing."
