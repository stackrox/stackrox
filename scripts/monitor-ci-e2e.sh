#!/bin/bash
# Monitor UI e2e tests for PR 20165
# Usage: ./scripts/monitor-ci-e2e.sh

COMMIT="e256edf2d3"
PR="20165"

echo "Monitoring commit ${COMMIT} (PR #${PR})"
echo "Press Ctrl+C to stop"
echo ""

while true; do
    clear
    echo "=== CI Status for commit ${COMMIT} ==="
    echo "Time: $(date '+%H:%M:%S')"
    echo ""

    # Get UI e2e tests
    TESTS=$(gh api repos/stackrox/stackrox/commits/${COMMIT}/check-runs \
        --jq '.check_runs[] | select(.name | contains("ui-e2e")) | {name, status, conclusion}' 2>/dev/null)

    if [ -z "$TESTS" ]; then
        echo "⏳ UI e2e tests not yet queued"
        echo ""
        echo "Other running tests:"
        gh api repos/stackrox/stackrox/commits/${COMMIT}/check-runs \
            --jq '.check_runs[] | select(.status == "in_progress") | "  - \(.name): \(.status)"' 2>/dev/null | head -5
    else
        echo "UI e2e tests:"
        echo "$TESTS" | jq -r '"  - \(.name): \(.status) - \(.conclusion // "in progress")"'

        # Check if all completed
        PENDING=$(echo "$TESTS" | jq -r 'select(.status != "completed") | .name' | wc -l)
        if [ "$PENDING" -eq 0 ]; then
            echo ""
            echo "✅ All UI e2e tests completed!"

            # Show results
            FAILED=$(echo "$TESTS" | jq -r 'select(.conclusion == "failure") | .name')
            if [ -z "$FAILED" ]; then
                echo "✅ All tests PASSED!"
            else
                echo "❌ Failed tests:"
                echo "$FAILED"
            fi
            break
        fi
    fi

    echo ""
    echo "Full status: https://github.com/stackrox/stackrox/pull/${PR}/checks"
    sleep 30
done
