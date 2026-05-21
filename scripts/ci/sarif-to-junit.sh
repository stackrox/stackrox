#!/usr/bin/env bash

# sarif-to-junit.sh converts SARIF vulnerability scan results to JUnit XML format
# for integration with junit2jira. This enables individual vulnerability tracking
# in Jira rather than just job-level failure notifications.
#
# Usage: sarif-to-junit.sh <sarif-file> <image-name>

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$SCRIPTS_ROOT/scripts/ci/lib.sh"

set -euo pipefail

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

# SARIF structure: .runs[].results[] contains vulnerability findings
# Each result has ruleId (CVE_component_version), message, and level (severity)

testcases_xml=""
vuln_count=0
failure_count=0

while IFS=$'\t' read -r rule_id level message location; do
    if [[ -z "$rule_id" ]]; then
        continue
    fi

    ((vuln_count++)) || true

    if [[ "$rule_id" =~ ^([^_]+)_(.+)$ ]]; then
        cve_id="${BASH_REMATCH[1]}"
        component_version="${BASH_REMATCH[2]}"
    else
        cve_id="$rule_id"
        component_version="unknown"
    fi

    failure_details="Vulnerability: ${rule_id}
Severity: ${level}
Component: ${location}
Image: ${image_name}

${message}"

    testcases_xml+="    <testcase name=\"${component_version}\" classname=\"${cve_id}\">
      <failure><![CDATA[${failure_details}]]></failure>
    </testcase>
"
    ((failure_count++)) || true
done < <(jq -r '.runs[].results[] | [.ruleId, .level, (.message.text // ""), (.locations[0].physicalLocation.artifactLocation.uri // "")] | @tsv' "$sarif_file")

junit_dir="$(get_junit_misc_dir)"
mkdir -p "${junit_dir}"

suite_name="${image_name}"
safe_filename="$(echo "${image_name}" | tr '/:' '_')"
junit_file="${junit_dir}/junit-${safe_filename}.xml"

if [[ $vuln_count -eq 0 ]]; then
    cat > "${junit_file}" <<EOF
<testsuite name="${suite_name}" tests="1" skipped="0" failures="0" errors="0">
  <testcase name="vulnerability-scan" classname="vulnerability-scan">
  </testcase>
</testsuite>
EOF
    echo "No vulnerabilities found in SARIF report"
else
    cat > "${junit_file}" <<EOF
<testsuite name="${suite_name}" tests="${vuln_count}" skipped="0" failures="${failure_count}" errors="0">
${testcases_xml}</testsuite>
EOF
    echo "Converted ${vuln_count} vulnerabilities from SARIF to JUnit"
fi
