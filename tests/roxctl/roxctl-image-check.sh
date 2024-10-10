#! /usr/bin/env bash

# This test script requires ROX_ADMIN_PASSWORD to be set in the environment.

[ -n "$ROX_ADMIN_PASSWORD" ]

FAILURES=0

eecho() {
  echo "$@" >&2
}

test_roxctl_cmd() {
  echo "Testing command: roxctl image check --image " "$@"

  # Verify image check.
  if OUTPUT=$(roxctl --insecure-skip-tls-verify --insecure image check --image \
    "$@" \
    2>&1); then
      echo "[OK] roxctl image check " "$@" " works"
  else
      eecho "[FAIL] roxctl image check " "$@" " fails"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  fi
}

test_roxctl_cmd nginx
test_roxctl_cmd nginx -c 'Docker CIS'

if [ $FAILURES -eq 0 ]; then
  echo "Passed"
else
  echo "$FAILURES test failed"
  exit 1
fi
