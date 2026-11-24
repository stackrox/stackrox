#!/usr/bin/env bash
# Watch PR checks; exit as soon as the set of terminal (pass/fail) results
# changes, so the caller can react to the newest result immediately.
#
# The first poll establishes the baseline (possibly empty, e.g. when no checks
# are running yet). Every later change in the set of terminal results is
# printed as a delta of newly terminal checks and the script exits 0.
# Exits 1 on timeout, 2 on usage errors.
#
# Usage:
#   watch-checks.sh <PR-number> [--repo <owner/name>] [--only <substring>]
#                   [--interval <seconds>] [--timeout <minutes>]
set -u

REPO="stackrox/stackrox"
ONLY=""
INTERVAL=90
TIMEOUT=360

usage() {
  cat <<EOF
Usage: $(basename "$0") <PR-number> [options]

Watch the checks of a GitHub pull request and exit as soon as a new check
reaches a terminal (pass/fail) state, printing only the new result(s).

Options:
  --repo <owner/name>   GitHub repository (default: ${REPO})
  --only <substring>    only consider checks whose name contains this substring
  --interval <seconds>  poll interval in seconds (default: ${INTERVAL})
  --timeout <minutes>   give up after this many minutes (default: ${TIMEOUT})
  -h, --help            show this help
EOF
}

is_positive_integer() {
  case "${1:-}" in
    '' | *[!0-9]*) return 1 ;;
  esac
  [ "$1" -gt 0 ]
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ $# -lt 1 ]; then
  usage >&2
  exit 2
fi

PR_NUMBER="$1"
shift

while [ $# -gt 0 ]; do
  # Every supported option takes a value; reject a dangling one up front.
  case "$1" in
    --repo | --only | --interval | --timeout)
      if [ $# -lt 2 ]; then
        echo "error: $1 requires a value" >&2
        usage >&2
        exit 2
      fi
      ;;
  esac
  case "$1" in
    --repo) REPO="$2" ;;
    --only) ONLY="$2" ;;
    --interval) INTERVAL="$2" ;;
    --timeout) TIMEOUT="$2" ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift 2
done

if ! is_positive_integer "$PR_NUMBER"; then
  echo "error: PR number must be a positive integer, got: ${PR_NUMBER}" >&2
  usage >&2
  exit 2
fi
if ! is_positive_integer "$INTERVAL"; then
  echo "error: --interval must be a positive integer, got: ${INTERVAL}" >&2
  exit 2
fi
if ! is_positive_integer "$TIMEOUT"; then
  echo "error: --timeout must be a positive integer, got: ${TIMEOUT}" >&2
  exit 2
fi

echo "Watching checks on ${REPO}#${PR_NUMBER} (interval ${INTERVAL}s, timeout ${TIMEOUT}m)"

# gh pr checks exits non-zero when any check has failed; capture output regardless.
poll_terminal_checks() {
  gh -R "$REPO" pr checks "$PR_NUMBER" 2>/dev/null |
    awk -F'\t' '$2 == "pass" || $2 == "fail" {print $1 "\t" $2}' |
    if [ -n "$ONLY" ]; then grep -- "$ONLY" || true; else cat; fi |
    sort
}

prev=""
baseline_set=0
deadline=$((TIMEOUT * 60))
SECONDS=0

while ((SECONDS < deadline)); do
  cur="$(poll_terminal_checks)"
  if [ "$baseline_set" -eq 0 ]; then
    prev="$cur"
    baseline_set=1
  elif [ -n "$cur" ] && [ "$cur" != "$prev" ]; then
    echo "=== NEW TERMINAL CHECK RESULT(S) ==="
    diff <(printf '%s\n' "$prev") <(printf '%s\n' "$cur") | grep '^>' | sed 's/^> //'
    exit 0
  fi
  sleep "$INTERVAL"
done

echo "Timed out after ${TIMEOUT}m with no new terminal results"
exit 1
