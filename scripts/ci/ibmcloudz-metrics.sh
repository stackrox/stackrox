#!/usr/bin/env bash
set -euo pipefail

# Collect IBM Cloud Z provisioning metrics
# Addresses: ROX-21457 - monitoring and alerting for failure tracking

METRICS_FILE="${ARTIFACT_DIR:-/tmp}/ibmcloudz-metrics.json"

collect_provisioning_metrics() {
    local start_time="$1"
    local end_time="$2"
    local status="$3"

    local duration=$((end_time - start_time))

    cat > "$METRICS_FILE" << EOF
{
  "job_name": "${JOB_NAME:-unknown}",
  "build_id": "${BUILD_ID:-unknown}",
  "cluster_arch": "s390x",
  "provision_start_epoch": $start_time,
  "provision_end_epoch": $end_time,
  "provision_duration_seconds": $duration,
  "provision_duration_minutes": $((duration / 60)),
  "status": "$status",
  "bootstrap_timeout": "${OPENSHIFT_INSTALL_BOOTSTRAP_TIMEOUT:-unknown}",
  "api_timeout": "${OPENSHIFT_INSTALL_API_WAIT_TIMEOUT:-unknown}",
  "operator_timeout": "${OPERATOR_TIMEOUT:-unknown}",
  "openshift_version": "${OPENSHIFT_VERSION:-unknown}",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

    echo "Metrics saved to $METRICS_FILE"
    cat "$METRICS_FILE"
}

# Usage examples:
# Source this file and call collect_provisioning_metrics
#   start_time=$(date +%s)
#   # ... provisioning happens ...
#   end_time=$(date +%s)
#   source scripts/ci/ibmcloudz-metrics.sh
#   collect_provisioning_metrics "$start_time" "$end_time" "success"
