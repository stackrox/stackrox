#! /usr/bin/env bash

# This test script requires ROX_PASSWORD to be set in the environment.

[ -n "$ROX_PASSWORD" ]

FAILURES=0

eecho() {
  echo "$@" >&2
}

test_roxctl_cmd() {
  echo "Testing command: roxctl central whoami"

  # Verify central whoami using current k8s context.
  if OUTPUT=$(roxctl --insecure-skip-tls-verify -p "$ROX_PASSWORD" central whoami --use-current-k8s-context \
    2>&1); then
      echo "[OK] roxctl central whoami using current k8s context works"
  else
      eecho "[FAIL] roxctl central whoami using current k8s context fails"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  fi
}

test_roxctl_cmd

if [ $FAILURES -eq 0 ]; then
  echo "Passed"
else
  echo "$FAILURES test failed"
  exit 1
fi
