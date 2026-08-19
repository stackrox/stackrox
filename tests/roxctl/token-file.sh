#! /usr/bin/env bash

# This test script requires API_ENDPOINT and ROX_ADMIN_PASSWORD to be set in the environment.

[ -n "$API_ENDPOINT" ]
[ -n "$ROX_ADMIN_PASSWORD" ]

echo "Using API_ENDPOINT $API_ENDPOINT"

FAILURES=0

eecho() {
  echo "$@" >&2
}

die() {
  eecho "$@"
  exit 1
}

curl_cfg() { # Use built-in echo to not expose $2 in the process list.
  echo -n "$1 = \"${2//[\"\\]/\\&}\""
}

# token_file_has_token rejects jq's "null" for a missing .token field.
token_file_has_token() {
  local token
  token="$(cat "$TOKEN_FILE")"
  [ -n "$token" ] && [ "$token" != "null" ]
}

request_api_token() {
  # wait_for_api_token owns retries; curl --retry would multiply generate POSTs.
  curl -k -f \
    --config <(curl_cfg user "admin:$ROX_ADMIN_PASSWORD") \
    -d '{"name": "test", "role": "Admin"}' \
    --connect-timeout 10 \
    --max-time 30 \
    "https://$API_ENDPOINT/v1/apitokens/generate"
}

write_token_file() {
  local response
  response="$(request_api_token)" || return 1
  printf '%s' "$response" | jq -r .token > "$TOKEN_FILE" || return 1
  token_file_has_token
}

# sleep_before_next_attempt waits 1s, 2s, 4s, then 8s after attempts 1-4.
sleep_before_next_attempt() {
  case "$1" in
    1) sleep 1 ;;
    2) sleep 2 ;;
    3) sleep 4 ;;
    4) sleep 8 ;;
  esac
}

wait_for_api_token() {
  local attempt=1
  local max_attempts=5

  while [ "$attempt" -le "$max_attempts" ]; do
    if write_token_file; then
      return 0
    fi
    if [ "$attempt" -eq "$max_attempts" ]; then
      die "Failed to retrieve API token after ${max_attempts} attempts: token file is empty"
    fi
    eecho "Token generate attempt ${attempt}/${max_attempts} failed, retrying..."
    sleep_before_next_attempt "$attempt"
    attempt=$((attempt + 1))
  done
}

TOKEN_FILE=$(mktemp)
wait_for_api_token

test_roxctl_cmd() {
  echo "Testing command: roxctl " "$@"
  local password="$ROX_ADMIN_PASSWORD"
  # Clear values for the test run.
  local ROX_ADMIN_PASSWORD=""
  local ROX_API_TOKEN_FILE=""
  local ROX_API_TOKEN=""

  # Verify that specifying a token file works.
  if OUTPUT=$(roxctl --ca "" --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
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
  if OUTPUT=$(roxctl --ca "" --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    --token-file "$TOKEN_FILE" \
    --password "secret" \
    "$@" \
    2>&1); then
      eecho "[FAIL] Specifying --token-file and --password did not produce error"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  elif echo "$OUTPUT" | grep -q "\[password token-file\] were all set"; then
    echo "[OK] Specifying --token-file and --password produced expected error message"
  else
    eecho "[FAIL] Specifying --token-file and --password produced error did not produce expected error message"
    eecho "Captured output was:"
    eecho "$OUTPUT"
    FAILURES=$((FAILURES + 1))
  fi

  # Verify that specifying a token file and password at the same time fails.
  # shellcheck disable=SC2030
  if OUTPUT=$(export ROX_API_TOKEN_FILE="$TOKEN_FILE" ROX_ADMIN_PASSWORD="$password"; \
    roxctl --ca "" --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    "$@" \
    2>&1); then
      eecho "[FAIL] Specifying ROX_API_TOKEN_FILE and ROX_ADMIN_PASSWORD did not produce error"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  elif echo "$OUTPUT" | grep -q "cannot use basic and token-based authentication at the same time"; then
    echo "[OK] Specifying ROX_API_TOKEN_FILE and ROX_ADMIN_PASSWORD produced expected error message"
  else
    eecho "[FAIL] Specifying ROX_API_TOKEN_FILE and ROX_ADMIN_PASSWORD did not produce expected error message"
    eecho "Captured output was:"
    eecho "$OUTPUT"
    FAILURES=$((FAILURES + 1))
  fi

  # Verify that token on the command line has precedence over token in the environment
  # shellcheck disable=SC2030
  if OUTPUT=$(export ROX_API_TOKEN="invalid-token"; \
    roxctl --ca "" --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
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
  # shellcheck disable=SC2031
  if OUTPUT=$(export ROX_API_TOKEN="invalid-token"; \
    roxctl --ca "" --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    --password "$password" \
    "$@" \
    2>&1); then
      echo "[OK] --password has precedence over ROX_API_TOKEN environment variable"
  else
      eecho "[FAIL] Invalid token in ROX_API_TOKEN causes failure even though valid password specified with --password"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  fi

  # Verify that a password on the command line has precedence over password in the environment.
  # shellcheck disable=SC2030,SC2031
  if OUTPUT=$(export ROX_ADMIN_PASSWORD="bad-password"; \
    roxctl --ca "" --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    --password "$password" \
    "$@" \
    2>&1); then
      echo "[OK] --password has precedence over ROX_ADMIN_PASSWORD environment variable"
  else
      eecho "[FAIL] Invalid password in ROX_ADMIN_PASSWORD causes failure even though valid password specified with --password"
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

  if OUTPUT=$(roxctl --ca "" --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
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

  # Verify that a password on the command line has precedence over token file in the environment.
  # shellcheck disable=SC2031
  if OUTPUT=$(export ROX_API_TOKEN_FILE="$TOKEN_FILE"; \
    roxctl --ca "" --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    --password "$password" \
    "$@" \
    2>&1); then
      echo "[OK] --password has precedence over ROX_API_TOKEN_FILE environment variable"
  else
      eecho "[FAIL] Invalid file in ROX_API_TOKEN_FILE causes failure even though valid password specified with --password"
      eecho "Captured output was:"
      eecho "$OUTPUT"
      FAILURES=$((FAILURES + 1))
  fi

  # Verify that the token file on the command line has precedence over password in the environment.
  # shellcheck disable=SC2031
  if OUTPUT=$(export ROX_ADMIN_PASSWORD="bad-password"; \
    roxctl --ca "" --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    --token-file "$TOKEN_FILE" \
    "$@" \
    2>&1); then
      echo "[OK] --token-file has precedence over ROX_ADMIN_PASSWORD environment variable"
  else
      eecho "[FAIL] Invalid password in ROX_ADMIN_PASSWORD causes failure even though valid token file specified with --token-file"
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
