#! /usr/bin/env bash

# This test script requires API_ENDPOINT and ROX_PASSWORD to be set in the environment.

[ -n "$API_ENDPOINT" ]
[ -n "$ROX_PASSWORD" ]

echo "Using API_ENDPOINT $API_ENDPOINT"

FAILURES=0

eecho() {
  echo "$@" >&2
}

# Retrieve API token
TOKEN_FILE=$(mktemp)
curl -k -f \
  -u "admin:$ROX_PASSWORD" \
  -d '{"name": "test", "role": "Admin"}' \
  "https://$API_ENDPOINT/v1/apitokens/generate" \
  | jq -r .token \
  > "$TOKEN_FILE"

[ -n "$(cat "$TOKEN_FILE")" ]

test_roxctl_cmd() {
  echo "Testing command: roxctl " "$@"

  # Verify that specifying a token file works.
  if OUTPUT=$(roxctl --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    --token-file "$TOKEN_FILE" \
    "$@" \
    2>&1); then
      echo "[OK] Specifying only --token-file works"
  else
      eecho "[FAIL] Specifying only --token-file fails"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  fi

  # Verify that specifying a token file and password at the same time fails.
  if OUTPUT=$(roxctl --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    --token-file "$TOKEN_FILE" \
    --password "secret" \
    "$@" \
    2>&1); then
      eecho "[FAIL] Specifying --token-file and --password did not produce error"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  elif echo "$OUTPUT" | grep -q "cannot use basic and token-based authentication at the same time"; then
    echo "[OK] Specifying --token-file and --password produced expected error message"
  else
    eecho "[FAIL] Specifying --token-file and --password produced error did not produce expected error message"
    eecho "Captured output was:"
    eecho "$OUTPUT"
    FAILURES=$((FAILURES + 1))
  fi

  # Verify that token on the command line has precedence over token in the environment
  if OUTPUT=$(ROX_API_TOKEN="invalid-token" roxctl --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    --token-file "$TOKEN_FILE" \
    "$@" \
    2>&1); then
      echo "[OK] --token-file has precedence over ROX_API_TOKEN environment variable"
  else
      eecho "[FAIL] Invalid token in ROX_API_TOKEN causes failure even though valid token specified with --token-file"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  fi

  # Verify that a password on the command line has precedence over token in the environment
  if OUTPUT=$(ROX_API_TOKEN="invalid-token" roxctl --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    --password "$ROX_PASSWORD" \
    "$@" \
    2>&1); then
      echo "[OK] --password has precedence over ROX_API_TOKEN environment variable"
  else
      eecho "[FAIL] Invalid token in ROX_API_TOKEN causes failure even though valid password specified with --password"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  fi

  # Verify that an invalid file specified with --token-file produces a hint to use ROX_API_TOKEN.
  NON_EXISTING="a-non-existing-file-without-slashes"
  if [ -e "$NON_EXISTING" ]; then
    eecho "This should not happen: a file with the made up name '$NON_EXISTING' exists unexpectedly"
    exit 1
  fi

  if OUTPUT=$(roxctl --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    --token-file "$NON_EXISTING" \
    "$@" \
    2>&1); then
      eecho "[FAIL] Specifying invalid file with --token-file succeeded"
      FAILURES=$((FAILURES + 1))
  elif echo "$OUTPUT" | grep -q "failed to retrieve token from file"; then
    echo "[OK] Specifying invalid file with --token-file produces error message with expected output"
  else
    eecho "[FAIL] Specifying invalid file with --token-file does not produce error message with expected output"
    eecho "Captured output was:"
    eecho "$OUTPUT"
    FAILURES=$((FAILURES + 1))
  fi
}

test_roxctl_cmd central whoami
test_roxctl_cmd central db backup

if [ $FAILURES -eq 0 ]; then
  echo "Passed"
else
  echo "$FAILURES test failed"
  exit 1
fi
