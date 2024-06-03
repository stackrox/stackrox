#! /usr/bin/env bash

# This test script requires ROX_PASSWORD to be set in the environment.

[ -n "$ROX_PASSWORD" ]

FAILURES=0

eecho() {
  echo "$@" >&2
}

test_roxctl_cmd() {
  echo "Testing command: roxctl central whoami"
  echo "Namespaces:"
  kubectl get ns
  echo "Switching the context to the stackrox namespace..."
  kubectl config set-context --current --namespace stackrox
  echo "Kube contexts:"
  kubectl config get-contexts
  echo "Central service endpoints:"
  kubectl get ep central

  echo "Creating a service account with a restricted role..."
  kubectl apply -f "../testdata/port-forward-role.yaml"
  echo "Switching the context to use the service account token..."
  TOKEN=$(kubectl get secret "port-forward-sa-secret" -o jsonpath='{.data.token}' | base64 --decode)
  kubectl config set-credentials port-forward-user --token="$TOKEN"
  CURRENT_CONTEXT=$(kubectl config current-context)
  CURRENT_CLUSTER=$(kubectl config view -o jsonpath="{.contexts[?(@.name=='$CURRENT_CONTEXT')].context.cluster}")
  kubectl config set-context port-forward-context --user="port-forward-user" --cluster="$CURRENT_CLUSTER"
  kubectl config use-context port-forward-context
  echo "New context:" "$(kubectl config current-context)"

  # Verify central whoami using current k8s context.
  if OUTPUT=$(roxctl -p "$ROX_PASSWORD" central whoami --use-current-k8s-context \
    2>&1); then
      echo "[OK] roxctl central whoami using current k8s context works"
  else
      eecho "[FAIL] roxctl central whoami using current k8s context fails"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  fi
  echo "Switch to the original context..."
  kubectl config use-context "$CURRENT_CONTEXT"
}

test_roxctl_cmd

if [ $FAILURES -eq 0 ]; then
  echo "Passed"
else
  echo "$FAILURES test failed"
  exit 1
fi
