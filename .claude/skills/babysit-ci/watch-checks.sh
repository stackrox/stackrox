#!/usr/bin/env bash
# Watch PR checks; exit as soon as a check newly reaches a terminal (pass/fail)
# state, so the caller can react to the newest result immediately.
#
# The first poll establishes the baseline (possibly empty, e.g. when no checks
# are running yet); any checks that have already FAILED at that point are
# reported immediately so they can be triaged. The baseline is persisted to a
# state file, so a restarted watcher (this script is meant to be re-armed
# after every notification) resumes from the last reported snapshot instead of
# silently absorbing results that landed in between, reporting only newly
# terminal checks.
#
# A check going back to pending (e.g. a rerun of a failed check) updates the
# baseline without exiting; when it reaches a terminal state again, that
# result is reported. When no considered check is pending anymore, the
# watcher prints "=== ALL CHECKS TERMINAL ===" (with any new results) and
# exits so the caller can give a final summary.
#
# Poll failures (auth or network errors) are distinguished from empty results
# and retried; the script gives up after 5 consecutive failed polls.
#
# Exits 0 when new terminal results were printed or all checks are terminal,
# 1 on timeout, 2 on usage errors, 3 after repeated poll failures, 4 when the
# state file cannot be written.
#
# Usage:
#   watch-checks.sh <PR-number> [--repo <owner/name>] [--only <substring>]
#                   [--interval <seconds>] [--timeout <minutes>]
#                   [--state-file <path>]
set -u

REPO="stackrox/stackrox"
ONLY=""
INTERVAL=90
TIMEOUT=360
STATE_FILE=""
MAX_POLL_FAILURES=5

usage() {
  cat <<EOF
Usage: $(basename "$0") <PR-number> [options]

Watch the checks of a GitHub pull request and exit as soon as a check newly
reaches a terminal (pass/fail) state, printing only the new result(s).

Options:
  --repo <owner/name>   GitHub repository (default: ${REPO})
  --only <substring>    only consider checks whose name contains this substring
  --interval <seconds>  poll interval in seconds (default: ${INTERVAL})
  --timeout <minutes>   give up after this many minutes (default: ${TIMEOUT})
  --state-file <path>   snapshot file for restart safety
                        (default: \${TMPDIR:-/tmp}/watch-checks-<repo>-<pr>.state)
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
    --repo | --only | --interval | --timeout | --state-file)
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
    --state-file) STATE_FILE="$2" ;;
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
  usage >&2
  exit 2
fi
if ! is_positive_integer "$TIMEOUT"; then
  echo "error: --timeout must be a positive integer, got: ${TIMEOUT}" >&2
  usage >&2
  exit 2
fi

if [ -z "$STATE_FILE" ]; then
  STATE_FILE="${TMPDIR:-/tmp}/watch-checks-${REPO//\//-}-${PR_NUMBER}.state"
fi

echo "Watching checks on ${REPO}#${PR_NUMBER} (interval ${INTERVAL}s, timeout ${TIMEOUT}m)"

ERR_FILE="$(mktemp)"
trap 'rm -f "$ERR_FILE"' EXIT

# Populates RAW with the full `gh pr checks` output (all states) and returns 0
# on a successful poll. `gh pr checks` exits non-zero when any check has
# failed; that is still a successful poll. An empty output combined with the
# "no checks reported" error is a valid empty result; any other error means
# the query itself failed and is reported as a poll failure.
RAW=""
poll_checks() {
  RAW="$(gh -R "$REPO" pr checks "$PR_NUMBER" 2>"$ERR_FILE")" || true
  if [ -n "$RAW" ]; then
    return 0
  fi
  if grep -q "no checks reported on the" "$ERR_FILE" 2>/dev/null; then
    return 0
  fi
  return 1
}

terminal_checks() {
  printf '%s\n' "$RAW" |
    awk -F'\t' '$2 == "pass" || $2 == "fail" {print $1 "\t" $2}' |
    if [ -n "$ONLY" ]; then grep -- "$ONLY" || true; else cat; fi |
    sort
}

# Whether any considered check is still pending. Requires a non-empty RAW (a
# successful poll can also be an empty "no checks reported" result, which must
# not be mistaken for a completed board).
has_pending() {
  printf '%s\n' "$RAW" |
    awk -F'\t' '$2 == "pending" {print $1}' |
    if [ -n "$ONLY" ]; then grep -- "$ONLY" || true; else cat; fi |
    grep -q .
}

all_terminal() {
  [ -n "$RAW" ] && ! has_pending
}

snapshot_lines() {
  if [ -n "$1" ]; then printf '%s\n' "$1"; fi
}

added_terminal() {
  diff <(snapshot_lines "$prev") <(snapshot_lines "$1") | grep '^>' | sed 's/^> //'
}

# Persist the snapshot so a restarted watcher resumes from it. A failed save
# would silently break restart safety, so treat it as fatal.
save_state() {
  if ! printf '%s' "$1" >"$STATE_FILE" 2>/dev/null; then
    echo "error: cannot write state file ${STATE_FILE}; restart safety is broken" >&2
    exit 4
  fi
}

prev=""
baseline_set=0

# Resume from a previous watcher's snapshot so checks that went terminal
# between watchers are still reported. The snapshot may legitimately be empty
# (all checks were pending when it was saved), so test for existence, not size.
if [ -f "$STATE_FILE" ]; then
  prev="$(cat "$STATE_FILE")"
  baseline_set=1
  echo "Resuming from snapshot in ${STATE_FILE}"
fi

consecutive_failures=0
deadline=$((TIMEOUT * 60))
SECONDS=0

while ((SECONDS < deadline)); do
  if ! poll_checks; then
    consecutive_failures=$((consecutive_failures + 1))
    echo "warning: gh pr checks failed (${consecutive_failures}/${MAX_POLL_FAILURES}): $(cat "$ERR_FILE")" >&2
    if [ "$consecutive_failures" -ge "$MAX_POLL_FAILURES" ]; then
      echo "error: giving up after ${MAX_POLL_FAILURES} consecutive failed polls" >&2
      exit 3
    fi
    sleep "$INTERVAL"
    continue
  fi
  consecutive_failures=0

  cur="$(terminal_checks)"
  if [ "$baseline_set" -eq 0 ]; then
    prev="$cur"
    baseline_set=1
    save_state "$prev"
    # Fresh start (no state file): surface failures that are already terminal
    # so they get triaged instead of silently becoming part of the baseline.
    # A resumed watcher does not do this; it reports only results that are new
    # since its saved snapshot.
    fails="$(printf '%s\n' "$cur" | awk -F'\t' '$2 == "fail" {print $1 "\t" $2}')"
    if [ -n "$fails" ]; then
      echo "=== ALREADY-TERMINAL FAILURE(S) AT WATCHER START ==="
      echo "$fails"
      if all_terminal; then
        echo "=== ALL CHECKS TERMINAL ==="
      fi
      exit 0
    fi
  else
    added="$(added_terminal "$cur")"
    if [ -n "$added" ]; then
      echo "=== NEW TERMINAL CHECK RESULT(S) ==="
      echo "$added"
      save_state "$cur"
      if all_terminal; then
        echo "=== ALL CHECKS TERMINAL ==="
      fi
      exit 0
    fi
    # No new terminal results: adopt the current snapshot (it may have lost
    # rows, e.g. a check going back to pending for a rerun) and keep waiting.
  fi
  # Signal completion when nothing is pending anymore, so the caller can give
  # a final summary instead of waiting for the timeout (this also covers a
  # fresh watcher starting against an already-complete board).
  if all_terminal; then
    echo "=== ALL CHECKS TERMINAL ==="
    save_state "$cur"
    exit 0
  fi
  prev="$cur"
  save_state "$prev"
  sleep "$INTERVAL"
done

echo "Timed out after ${TIMEOUT}m with no new terminal results (snapshot kept in ${STATE_FILE})"
exit 1
