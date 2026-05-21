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

if [[ -z "${ARTIFACT_DIR:-}" ]]; then
    die "ARTIFACT_DIR environment variable must be set"
fi

# Extract vulnerabilities from SARIF and build JUnit XML
# SARIF structure: .runs[].results[] contains vulnerability findings
# Each result has ruleId (CVE_component_version), message, and level (severity)

# Collect all test cases
declare -a testcases=()
vuln_count=0
failure_count=0

while IFS=$'\t' read -r rule_id level message location; do
    if [[ -z "$rule_id" ]]; then
        continue
    fi

    ((vuln_count++)) || true

    # Parse rule_id: "CVE-2024-1234_nginx_1.19.0" -> CVE="CVE-2024-1234", component="nginx_1.19.0"
    # Use the full rule_id as fallback if parsing fails
    cve_id="${rule_id%%_*}"
    component_version="${rule_id#*_}"

    # If parsing didn't split anything, use rule_id as CVE and "unknown" as component
    if [[ "$cve_id" == "$rule_id" ]]; then
        cve_id="$rule_id"
        component_version="unknown"
    fi

    # Build detailed failure message with vulnerability context
    failure_details=$(cat <<EOF
Vulnerability: ${rule_id}
Severity: ${level}
Component: ${location}
Image: ${image_name}

${message}
EOF
    )

    # XML-escape the failure details (basic escaping for CDATA)
    # CDATA can't contain ]]> so we don't need complex escaping, just use CDATA

    testcases+=("$(cat <<EOF
    <testcase name="${component_version}" classname="${cve_id}">
      <failure><![CDATA[${failure_details}]]></failure>
    </testcase>
EOF
    )")
    ((failure_count++)) || true
done < <(jq -r '.runs[].results[] | [.ruleId, .level, (.message.text // ""), (.locations[0].physicalLocation.artifactLocation.uri // "")] | @tsv' "$sarif_file")

# Generate JUnit XML
junit_dir="$(get_junit_misc_dir)"
mkdir -p "${junit_dir}"

# Use image name as testsuite name (sanitize for filename)
suite_name="${image_name}"
safe_filename="$(echo "${image_name}" | tr '/:' '_')"
junit_file="${junit_dir}/junit-${safe_filename}.xml"

if [[ $vuln_count -eq 0 ]]; then
    # No vulnerabilities found - create success test suite
    cat > "${junit_file}" <<EOF
<testsuite name="${suite_name}" tests="1" skipped="0" failures="0" errors="0">
  <testcase name="vulnerability-scan" classname="vulnerability-scan">
  </testcase>
</testsuite>
EOF
    echo "No vulnerabilities found in SARIF report"
else
    # Create test suite with all vulnerability test cases
    cat > "${junit_file}" <<EOF
<testsuite name="${suite_name}" tests="${vuln_count}" skipped="0" failures="${failure_count}" errors="0">
$(printf '%s\n' "${testcases[@]}")
</testsuite>
EOF
    echo "Converted ${vuln_count} vulnerabilities from SARIF to JUnit"
fi
