#! /usr/bin/env bash

# This test script requires ROX_ADMIN_PASSWORD to be set in the environment.

[ -n "$ROX_ADMIN_PASSWORD" ]

FAILURES=0

eecho() {
  echo "$@" >&2
}

test_roxctl_cmd() {
  echo "Testing command: roxctl central whoami"
  CURRENT_CONTEXT=$(kubectl config current-context)
  CURRENT_CLUSTER=$(kubectl config view -o jsonpath="{.contexts[?(@.name=='$CURRENT_CONTEXT')].context.cluster}")

  echo "Namespaces:"
  kubectl get ns
  echo "Switching the context to the stackrox namespace..."
  kubectl config set-context --current --namespace stackrox
  echo "Kube contexts:"
  kubectl config get-contexts
  echo "Central service endpoints:"
  kubectl get ep central

  echo "Creating a service account with an insufficient role..."
  kubectl apply -f "tests/testdata/port-forward-role-bad.yaml"
  kubectl apply -f "tests/testdata/port-forward-sa.yaml"
  echo "Switching the context to use the service account token..."
  TOKEN=$(kubectl create token port-forward-sa)
  kubectl config set-credentials port-forward-user --token="$TOKEN"
  kubectl config set-context port-forward-context --user="port-forward-user" --cluster="$CURRENT_CLUSTER" --namespace="stackrox"
  kubectl config use-context port-forward-context
  echo "New context:" "$(kubectl config current-context)"

  # Verify central whoami using current k8s context.
  OUTPUT=$(roxctl central whoami --use-current-k8s-context 2>&1) || true
  if [[ "$OUTPUT" == "ERROR:"*"could not get endpoint"*"cannot list resource"* ]] ; then
      echo "[OK] roxctl central whoami using current k8s context and insufficient role fails"
  else
      eecho "[FAIL] roxctl central whoami using current k8s context and insufficient role works"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  fi

  echo "Switch to the original context..."
  kubectl config use-context "$CURRENT_CONTEXT"

  echo "Updating the role with sufficient permissions..."
  kubectl apply -f "tests/testdata/port-forward-role-minimal.yaml"

  echo "Switching back to the limited context..."
  kubectl config use-context port-forward-context
  # Verify central whoami using current k8s context.
  if OUTPUT=$(roxctl central whoami --use-current-k8s-context \
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
