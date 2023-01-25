#! /usr/bin/env bash

set -uo pipefail

roxctl helm >/dev/null 2>&1 || {
  echo "'roxctl helm' command unavailable, skipping test"
  exit 0
}

# This test script requires MAIN_TAG.
[ -n "$MAIN_TAG" ]

FAILURES=0

eecho() {
  echo "$@" >&2
}

die() {
    eecho "$@"
    exit 1
}


test_central_services_chart_generation() {
    local chart_name="central-services"
    local output_dir="$(mktemp -d)"
    local chart_output_dir="$output_dir/uncharted"

    if OUTPUT="$(roxctl helm output "$chart_name" --output-dir="$chart_output_dir" 2>&1)"; then
        echo "[OK] 'roxctl helm output $chart_name' succeeded"
    else
        eecho "[FAIL] 'roxctl helm output $chart_name' failed:"
        eecho "$OUTPUT"
        FAILURES=$((FAILURES + 1))
    fi

    if [ -e "$chart_output_dir/Chart.yaml" ] && grep -q "^appVersion: $MAIN_TAG" "$chart_output_dir/Chart.yaml"; then
        echo "[OK] Chart.yaml:appVersion is '$MAIN_TAG'"
    else
        FAILURES=$((FAILURES + 1))
        if [ -e "$chart_output_dir/Chart.yaml" ]; then
            app_version="$(grep "^appVersion:" "$chart_output_dir/Chart.yaml")"
            eecho "[FAIL] Chart.yaml:appVersion is not '$MAIN_TAG'. Chart.yaml contains: '$app_version'"
        else
            eecho "[FAIL] Chart.yaml not generated"
        fi
    fi
}

test_central_services_chart_generation

if [ $FAILURES -eq 0 ]; then
  echo "Passed"
else
  echo "$FAILURES tests failed"
  exit 1
fi
