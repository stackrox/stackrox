#!/usr/bin/env bash

# TODO(ROX-8801): Move these tests to bats.

set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

# Check bash version (require 4.0+ for associative arrays)
if [[ "${BASH_VERSINFO[0]}" -lt 4 ]]; then
    echo "ERROR: This script requires Bash 4.0 or later (found ${BASH_VERSION})" >&2
    echo "  Associative arrays (declare -A) require Bash 4.0+" >&2
    echo "  Your bash: $(command -v bash) (version ${BASH_VERSION})" >&2
    exit 1
fi

extra_args=()
if [[ -n "${CA:-}" ]]; then
  extra_args+=(--ca "$CA")
else
  extra_args+=(--insecure-skip-tls-verify)
fi

# Add password if provided
if [[ -n "${ROX_PASSWORD:-}" ]]; then
  extra_args+=(-p "$ROX_PASSWORD")
fi

# Define test cases: file -> expected policy names (pipe-separated)
# Each file is tested against a list of expected policy violations
declare -A TEST_CASES=(
    ["cronjob.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["daemonset.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["deployment.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["deploymentconfig.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["job.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["legacy-deploymentconfig.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["list-deployment.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["multi-container-pod.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["multi-deployment-crd.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["multi-deployment-route.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["multi-deployment.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["pod.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["replicaset.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["replicationcontroller.yaml"]="Latest tag|No CPU request or memory limit specified"
    ["statefulset.yaml"]="Latest tag|No CPU request or memory limit specified"
)

FAILED="false"

for yaml_file in "${!TEST_CASES[@]}"; do
    yaml_path="$DIR/$yaml_file"

    if [[ ! -f "$yaml_path" ]]; then
        >&2 echo "ERROR: Expected file not found: $yaml_path"
        FAILED="true"
        continue
    fi

    expected_policies="${TEST_CASES[$yaml_file]}"
    IFS='|' read -ra expected_array <<< "$expected_policies"
    expected_count="${#expected_array[@]}"

    # Build jq filter for expected policies
    jq_filter=""
    for policy in "${expected_array[@]}"; do
        if [[ -z "$jq_filter" ]]; then
            jq_filter=".==\"$policy\""
        else
            jq_filter="$jq_filter or .==\"$policy\""
        fi
    done

    # Get alerts from roxctl
    alerts_json="$(roxctl "${extra_args[@]}" -e "$API_ENDPOINT" deployment check --file "$yaml_path" --output json 2>/dev/null)"

    # Extract matching policy names (new format uses .results[].violatedPolicies[])
    matching_policies="$(echo "$alerts_json" | jq -r ".results[].violatedPolicies[].name | select($jq_filter)" 2>/dev/null || echo "")"
    actual_count="$(echo "$matching_policies" | grep -c . || echo "0")"

    if [[ "$actual_count" != "$expected_count" ]]; then
        >&2 echo "FAILED: $yaml_file - expected $expected_count alert(s), found $actual_count"
        >&2 echo "  Expected policies: ${expected_array[*]}"
        if [[ -n "$matching_policies" ]]; then
            >&2 echo "  Found policies: $matching_policies"
        else
            >&2 echo "  Found policies: (none)"
        fi
        FAILED="true"
    else
        # Verify each expected policy was found
        all_found=true
        for expected_policy in "${expected_array[@]}"; do
            if ! echo "$matching_policies" | grep -Fq "$expected_policy"; then
                >&2 echo "FAILED: $yaml_file - missing expected policy: $expected_policy"
                all_found=false
            fi
        done
        if [[ "$all_found" == "true" ]]; then
            echo "Analyzed $yaml_file successfully (found $expected_count expected alert(s))"
        else
            FAILED="true"
        fi
    fi
done

if [[ "$FAILED" == "true" ]]; then
    echo "Roxctl test failed"
    exit 1
fi
echo "All roxctl yaml verification tests passed"
exit 0
