#!/usr/bin/env bash
set -euo pipefail

# Classify IBM Cloud Z failures for better tracking and trend analysis
# Addresses: ROX-21457 - failure categorization
#
# This script analyzes logs and classifies failures into known categories
# to help track which mitigations are working and where to focus efforts.

LOG_FILE="${1:-}"
CLASSIFICATION_FILE="${ARTIFACT_DIR:-/tmp}/failure-classification.txt"
CLASSIFICATION_DETAIL_FILE="${ARTIFACT_DIR:-/tmp}/failure-classification-detail.json"

if [ -z "$LOG_FILE" ] || [ ! -f "$LOG_FILE" ]; then
    echo "Usage: $0 <log_file>"
    echo "unknown" > "$CLASSIFICATION_FILE"
    exit 0
fi

# Initialize classification
CLASSIFICATION="UNKNOWN"
DESCRIPTION=""
EVIDENCE=""

# Check for each known failure pattern
if grep -q "gzip: invalid checksum" "$LOG_FILE"; then
    CLASSIFICATION="GZIP_CHECKSUM"
    DESCRIPTION="Corrupted RHCOS s390x image cache"
    EVIDENCE=$(grep -A 2 "gzip: invalid checksum" "$LOG_FILE" | head -5)

elif grep -q "Failed waiting for Kubernetes API" "$LOG_FILE"; then
    CLASSIFICATION="K8S_API_TIMEOUT"
    DESCRIPTION="Bootstrap VM provisioning timeout on s390x infrastructure"
    EVIDENCE=$(grep -A 5 "Failed waiting for Kubernetes API" "$LOG_FILE" | head -10)

elif grep -q "Bootstrap failed to complete" "$LOG_FILE"; then
    CLASSIFICATION="BOOTSTRAP_TIMEOUT"
    DESCRIPTION="Bootstrap process timeout (DNS/network issues)"
    EVIDENCE=$(grep -A 5 "Bootstrap failed to complete" "$LOG_FILE" | head -10)

elif grep -q "Cluster operator.*Degraded" "$LOG_FILE"; then
    CLASSIFICATION="OPERATOR_DEGRADED"
    DESCRIPTION="OpenShift cluster operator degraded during initialization"
    # Extract which operator(s) are degraded
    EVIDENCE=$(grep "Cluster operator.*Degraded" "$LOG_FILE" | head -5)

elif grep -q "user over quota\|Quota:" "$LOG_FILE"; then
    CLASSIFICATION="QUOTA_EXCEEDED"
    DESCRIPTION="IBM Cloud resource quota exceeded (vCPU/memory)"
    EVIDENCE=$(grep -B 2 -A 2 "quota" "$LOG_FILE" | head -10)

elif grep -qi "terraform.*error" "$LOG_FILE"; then
    CLASSIFICATION="TERRAFORM_ERROR"
    DESCRIPTION="Terraform provisioning error (IBM Cloud API)"
    EVIDENCE=$(grep -i "terraform.*error" "$LOG_FILE" | head -5)

elif grep -q "connection reset by peer" "$LOG_FILE"; then
    CLASSIFICATION="IBM_AUTH_CONNECTION_RESET"
    DESCRIPTION="IBM Cloud IAM authentication connection reset"
    EVIDENCE=$(grep "connection reset by peer" "$LOG_FILE" | head -5)

fi

# Write simple classification to file (for easy parsing)
echo "$CLASSIFICATION" > "$CLASSIFICATION_FILE"

# Write detailed classification to JSON
cat > "$CLASSIFICATION_DETAIL_FILE" << EOF
{
  "classification": "$CLASSIFICATION",
  "description": "$DESCRIPTION",
  "log_file": "$LOG_FILE",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "build_id": "${BUILD_ID:-unknown}",
  "job_name": "${JOB_NAME:-unknown}",
  "evidence_snippet": $(echo "$EVIDENCE" | jq -Rs . || echo '""')
}
EOF

echo "Failure classified as: $CLASSIFICATION"
echo "Description: $DESCRIPTION"
echo ""
echo "Classification written to:"
echo "  $CLASSIFICATION_FILE"
echo "  $CLASSIFICATION_DETAIL_FILE"

# Also output to console for immediate visibility
cat "$CLASSIFICATION_DETAIL_FILE"
