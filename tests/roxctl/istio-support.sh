#!/usr/bin/env bash

# This test script requires API_ENDPOINT and ROX_PASSWORD to be set in the environment.

[ -n "$API_ENDPOINT" ]
[ -n "$ROX_PASSWORD" ]

echo "Using API_ENDPOINT $API_ENDPOINT"

FAILURES=0

eecho() {
  echo "$@" >&2
}

test_roxctl_cmd() {
  echo "Testing command: roxctl " "$@"

  output_dir=$(mktemp -d)
  cmd=(roxctl --insecure-skip-tls-verify -e "$API_ENDPOINT" -p "$ROX_PASSWORD" "$@" --output-dir "$output_dir")

  echo "${cmd[@]}"

  rm -rf "$output_dir" 2>/dev/null || true
  # Verify that if no istio-support flag is specified, no destination rules are created
  if OUTPUT=$("${cmd[@]}" 2>&1); then
    if ! find "$output_dir" -name helm -prune -false -o -name '*.yaml' | xargs grep -q "DestinationRule" ; then
      echo "[OK] No DestinationRule will be created"
    else
      eecho "[FAIL] DestinationRules found in generated YAMLs"
      FAILURES=$((FAILURES + 1))
    fi
  else
      eecho "[FAIL] Error invoking command"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  fi

  rm -rf "$output_dir" 2>/dev/null || true
  # Verify that with an istio-support flag, destination rules are created
  if OUTPUT=$("${cmd[@]}" --istio-support=1.5 2>&1); then
    if find "$output_dir" -name helm -prune -false -o -name '*.yaml' | xargs grep -q "DestinationRule" ; then
      echo "[OK] DestinationRules found in generated YAMLs".
    else
      eecho "[FAIL] DestinationRules not found in generated YAMLs"
      FAILURES=$((FAILURES + 1))
    fi
  else
      eecho "[FAIL] Error invoking command"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  fi

  rm -rf "$output_dir" 2>/dev/null || true
}

test_roxctl_cmd central generate k8s none --output-format kubectl
test_roxctl_cmd central generate openshift none

test_roxctl_cmd sensor generate k8s --name k8s-istio-test-cluster  --continue-if-exists
test_roxctl_cmd sensor get-bundle k8s-istio-test-cluster
test_roxctl_cmd sensor generate openshift --name os-istio-test-cluster --continue-if-exists
test_roxctl_cmd sensor get-bundle os-istio-test-cluster

if [ $FAILURES -eq 0 ]; then
  echo "Passed"
else
  echo "$FAILURES test failed"
  exit 1
fi
