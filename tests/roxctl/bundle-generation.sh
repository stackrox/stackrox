#!/usr/bin/env bash

# This test script requires API_ENDPOINT and ROX_PASSWORD to be set in the environment.

set -eo pipefail

eecho() {
  echo "$@" >&2
}

die() {
  eecho "$@"
  exit 1
}

[[ -n "$API_ENDPOINT" ]] || die "API_ENDPOINT environment variable required"
[[ -n "$ROX_PASSWORD" ]] || die "ROX_PASSWORD environment variable required"

echo "Using API_ENDPOINT $API_ENDPOINT"

FAILURES=0

roxctl_cmd() {
  roxctl --insecure-skip-tls-verify -e "$API_ENDPOINT" -p "$ROX_PASSWORD" "$@"
}

test_roxctl_cmd() {
  echo "Testing command: roxctl " "$@"

  output_dir=$(mktemp -d)
  cmd=(roxctl_cmd "$@" --output-dir "$output_dir")

  echo "${cmd[@]}"

  rm -rf "$output_dir" 2>/dev/null || true
  if OUTPUT=$("${cmd[@]}" 2>&1); then
    echo "[OK] Obtained bundle"
  else
      eecho "[FAIL] Error invoking command"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  fi
  rm -rf "$output_dir" 2>/dev/null || true
}

test_roxctl_cmd sensor generate k8s --name k8s-test-cluster  --continue-if-exists
test_roxctl_cmd sensor get-bundle k8s-test-cluster
roxctl_cmd cluster delete --name k8s-test-cluster || true

test_roxctl_cmd sensor generate openshift --openshift-version 3 --name oc3-test-cluster  --continue-if-exists
test_roxctl_cmd sensor get-bundle oc3-test-cluster
roxctl_cmd cluster delete --name oc3-test-cluster || true

test_roxctl_cmd sensor generate openshift --openshift-version 4 --name oc4-test-cluster  --continue-if-exists
test_roxctl_cmd sensor get-bundle oc4-test-cluster
roxctl_cmd cluster delete --name oc4-test-cluster || true

[[ $FAILURES -eq 0 ]] || die "$FAILURES test(s) failed"

echo "Passed"
