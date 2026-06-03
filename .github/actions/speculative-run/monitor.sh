#!/bin/bash
# Speculative-run background monitor.
# Polls the GHA check-runs API for sibling copies' in_progress/success signals.
# Environment (set by the gate action before nohup):
#   SPEC_REPO         github.repository
#   SPEC_CHECK_NAME   required check name (without copy letter)
#   SPEC_SIBLING_IDS  comma-separated external_ids to watch, e.g. "spec-a,spec-b"
#   SPEC_PARENT       PPID of the gate step's bash process (the runner worker)
#   GITHUB_SHA        provided by GHA automatically

sleep $(( RANDOM % 3 + 8 ))   # 8-10s: past checkout+gate, spreads API calls across copies
deadline=$(( $(date +%s) + 300 ))
poll=0
IFS=',' read -ra sibling_arr <<< "$SPEC_SIBLING_IDS"

while [[ $(date +%s) -lt $deadline ]]; do
  poll=$(( poll + 1 ))
  response=$(gh api "repos/${SPEC_REPO}/commits/${GITHUB_SHA}/check-runs?per_page=100" 2>/dev/null)

  winner=""
  all_gone=1

  for sid in "${sibling_arr[@]}"; do
    # Classify this sibling's check-run:
    #   winner  — in_progress or success (sibling's gate passed and is working/done)
    #   absent  — no check-run posted yet (sibling still in checkout, or was slow)
    #   pending — check posted but not yet complete (shouldn't happen with our flow)
    #   failed  — check completed with non-success (shouldn't happen; gate posts in_progress)
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

    # Keep waiting if: sibling is pending, or absent before poll 3.
    # Polls 1-3 cover the first ~12s from job start (checkout+gate window).
    # After poll 3, an absent sibling never posted → it was a slow runner → gone.
    if [[ "$result" == "pending" || ( "$result" == "absent" && $poll -lt 3 ) ]]; then
      all_gone=0
    fi
  done

  if [[ -n "$winner" ]]; then
    echo "Sibling $winner is winning — cancelling this copy."
    kill -TERM "$SPEC_PARENT" 2>/dev/null
    sleep 0.1
    new=$(comm -13 /tmp/speculative-baseline <(pgrep -P "$SPEC_PARENT" | sort) 2>/dev/null)
    for pid in $new; do kill -9 "$pid" 2>/dev/null; done
    break
  fi

  if [[ $all_gone -eq 1 ]]; then
    echo "All siblings gone (slow runners) — this copy proceeds."
    break
  fi

  sleep 2
done
