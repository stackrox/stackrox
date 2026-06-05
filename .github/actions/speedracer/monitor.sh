#!/bin/bash
# Speedracer background monitor.
# Polls the GHA check-runs API for sibling copies' in_progress/success signals.
#
# When a winner is found: SIGTERM the Gate step's composite action coordinator
# ($SPEC_PARENT). Since the Gate step is still alive (sleeping in a poll loop),
# the composite coordinator is still active — SIGTERM to it produces 'cancelled'.
#
# When all siblings are gone (slow runners): write a file to wake the Gate step
# so it exits 0 and allows work steps to proceed.
#
# Environment (set by the gate action before nohup):
#   SPEC_REPO         github.repository
#   SPEC_CHECK_NAME   required check name (without speedracer letter)
#   SPEC_SIBLING_IDS  comma-separated external_ids, e.g. "spec-a-123,spec-b-123"
#   SPEC_PARENT       PPID of the gate step's bash (composite action coordinator)
#   SPEC_SHA          github.sha

sleep $(( RANDOM % 2 + 5 ))   # 5-6s: past checkout+gate, spreads API calls
deadline=$(( $(date +%s) + 300 ))
poll=0
IFS=',' read -ra sibling_arr <<< "$SPEC_SIBLING_IDS"

while [[ $(date +%s) -lt $deadline ]]; do
  poll=$(( poll + 1 ))
  # filter=all returns EVERY check-run (not just the latest per name).
  response=$(gh api "repos/${SPEC_REPO}/commits/${SPEC_SHA}/check-runs?per_page=100&filter=all")

  winner=""
  all_gone=1

  for sid in "${sibling_arr[@]}"; do
    result=$(echo "$response" | jq -r --arg cn "$SPEC_CHECK_NAME" --arg sid "$sid" \
      '[.check_runs[] | select(.name == $cn and .external_id == $sid)] as $checks |
      if   ($checks | any(.[]; .status == "in_progress" or .conclusion == "success")) then "winner"
      elif ($checks | length == 0)                                                    then "absent"
      elif ($checks | any(.[]; .conclusion == null))                                  then "pending"
      else "failed" end')

    echo "[$(date -u +%H:%M:%S)] poll=$poll $sid: ${result:-absent}"

    if [[ "$result" == "winner" ]]; then
      winner="$sid"
      break
    fi

    if [[ "$result" == "pending" || ( "$result" == "absent" && $poll -lt 2 ) ]]; then
      all_gone=0
    fi
  done

  if [[ -n "$winner" ]]; then
    echo "Sibling $winner is winning — cancelling Gate step via SIGTERM to composite coordinator."
    kill -TERM "$SPEC_PARENT"
    break
  fi

  if [[ $all_gone -eq 1 ]]; then
    echo "All siblings gone (slow runners) — waking Gate step to proceed with work."
    echo "all_gone" > /tmp/speedracer-monitor-result
    break
  fi

  sleep 2
done

if [[ ! -f /tmp/speedracer-monitor-result ]]; then
  echo "Monitor deadline — waking Gate step."
  echo "timeout" > /tmp/speedracer-monitor-result
fi
