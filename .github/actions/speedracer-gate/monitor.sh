#!/bin/bash
# Speculative-run background monitor.
# Polls the GHA check-runs API for sibling copies' in_progress/success signals.
#
# When a winner is found: SIGTERM the Gate step's composite action coordinator
# ($SPEC_PARENT). Since the Gate step is still alive (sleeping in a poll loop),
# the composite coordinator is still active — SIGTERM to it produces 'cancelled'.
#
# When all siblings are gone (slow runners): write a file to wake up the Gate
# step's poll loop so it can exit 0 and allow work steps to proceed.
#
# Environment (set by the gate action before nohup):
#   SPEC_REPO         github.repository
#   SPEC_CHECK_NAME   required check name (without copy letter)
#   SPEC_SIBLING_IDS  comma-separated external_ids to watch, e.g. "spec-a,spec-b"
#   SPEC_PARENT       PPID of the gate step's bash (composite action coordinator)
#   SPEC_SHA          github.sha

sleep $(( RANDOM % 2 + 5 ))   # 5-6s: past checkout+gate, spreads API calls
deadline=$(( $(date +%s) + 300 ))
poll=0
IFS=',' read -ra sibling_arr <<< "$SPEC_SIBLING_IDS"

while [[ $(date +%s) -lt $deadline ]]; do
  poll=$(( poll + 1 ))
  # filter=all returns EVERY check-run (not just the latest per name).
  response=$(gh api "repos/${SPEC_REPO}/commits/${SPEC_SHA}/check-runs?per_page=100&filter=all" 2>/dev/null)

  winner=""
  all_gone=1

  for sid in "${sibling_arr[@]}"; do
    result=$(echo "$response" | jq -r --arg cn "$SPEC_CHECK_NAME" --arg sid "$sid" \
      '[.check_runs[] | select(.name == $cn and .external_id == $sid)] as $checks |
      if   ($checks | any(.[]; .status == "in_progress" or .conclusion == "success")) then "winner"
      elif ($checks | length == 0)                                                    then "absent"
      elif ($checks | any(.[]; .conclusion == null))                                  then "pending"
      else "failed" end' 2>/dev/null)

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
    # Gate step is alive (sleeping in a poll loop). SPEC_PARENT is the composite
    # action coordinator — SIGTERM to it produces 'cancelled' (not 'failure').
    kill -TERM "$SPEC_PARENT" 2>/dev/null
    # No child-kill needed: Gate step is just sleeping, no real work processes.
    break
  fi

  if [[ $all_gone -eq 1 ]]; then
    echo "All siblings gone (slow runners) — waking Gate step to proceed with work."
    echo "all_gone" > /tmp/speculative-monitor-result
    break
  fi

  sleep 2
done

# Deadline reached without a decision — wake Gate step to proceed.
if [[ ! -f /tmp/speculative-monitor-result ]]; then
  echo "Monitor deadline — waking Gate step."
  echo "timeout" > /tmp/speculative-monitor-result
fi
