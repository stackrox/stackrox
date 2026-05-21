#!/usr/bin/env bash

set -euo pipefail

# sarif-to-junit.sh converts SARIF vulnerability scan results to JUnit XML format
# for integration with junit2jira. This enables individual vulnerability tracking
# in Jira rather than just job-level failure notifications.
#
# Usage: sarif-to-junit.sh <sarif-file> <image-name>

source "$(dirname "$0")/lib.sh"

if [[ "$#" -ne 2 ]]; then
    die "Usage: $0 <sarif-file> <image-name>"
fi

sarif_file="$1"
image_name="$2"

if [[ ! -f "$sarif_file" ]]; then
    die "SARIF file not found: $sarif_file"
fi

# Extract vulnerabilities from SARIF and create JUnit records
# SARIF structure: .runs[].results[] contains vulnerability findings
# Each result has ruleId (CVE), message, and level (severity)

# Use process substitution to avoid subshell variable scope issues
vuln_count=0
while IFS=$'\t' read -r rule_id level message location; do
    if [[ -z "$rule_id" ]]; then
        continue
    fi

    # Build detailed failure message with vulnerability context
    details=$(cat <<EOF
Vulnerability: ${rule_id}
Severity: ${level}
Component: ${location}
Image: ${image_name}

${message}
EOF
    )

    # Use CVE ID as both class and description for junit2jira
    save_junit_failure "${rule_id}" "${rule_id}" "$details"
    ((vuln_count++)) || true
done < <(jq -r '.runs[].results[] | [.ruleId, .level, (.message.text // ""), (.locations[0].physicalLocation.artifactLocation.uri // "")] | @tsv' "$sarif_file")

if [[ $vuln_count -eq 0 ]]; then
    # No vulnerabilities found - record success
    save_junit_success "$(basename "$image_name")" "vulnerability scan"
    echo "No vulnerabilities found in SARIF report"
else
    echo "Converted ${vuln_count} vulnerabilities from SARIF to JUnit"
fi
