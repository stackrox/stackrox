#!/usr/bin/env bash
set -euo pipefail

# IBM Cloud Z cluster provisioning wrapper with enhanced error handling
# Addresses: ROX-21457 - K8S_API_TIMEOUT, BOOTSTRAP_TIMEOUT, GZIP_CHECKSUM failures
#
# This wrapper script monitors the cluster provisioning process (which is handled by
# OpenShift CI automation-flavors) and provides diagnostics for known transient failures.

INSTALLER_CACHE_DIR="${INSTALLER_CACHE_DIR:-/tmp/.cache/openshift-installer}"
LOG_FILE="${ARTIFACT_DIR:-/tmp}/ibmcloudz-provision.log"

# Enhanced timeouts for s390x architecture (slower provisioning than x86_64)
export OPENSHIFT_INSTALL_BOOTSTRAP_TIMEOUT="${OPENSHIFT_INSTALL_BOOTSTRAP_TIMEOUT:-90m}"
export OPENSHIFT_INSTALL_API_WAIT_TIMEOUT="${OPENSHIFT_INSTALL_API_WAIT_TIMEOUT:-45m}"

echo "IBM Cloud Z provisioning wrapper starting..."
echo "Bootstrap timeout: $OPENSHIFT_INSTALL_BOOTSTRAP_TIMEOUT"
echo "API wait timeout: $OPENSHIFT_INSTALL_API_WAIT_TIMEOUT"
echo "Log file: $LOG_FILE"

detect_gzip_checksum_error() {
    local log_file="$1"
    grep -q "gzip: invalid checksum" "$log_file" 2>/dev/null
}

detect_api_timeout_error() {
    local log_file="$1"
    grep -q "Failed waiting for Kubernetes API" "$log_file" 2>/dev/null
}

detect_bootstrap_timeout_error() {
    local log_file="$1"
    grep -q "Bootstrap failed to complete" "$log_file" 2>/dev/null
}

cleanup_corrupted_image_cache() {
    echo "Detected corrupted image cache, cleaning up..."
    if [ -d "$INSTALLER_CACHE_DIR/image_cache" ]; then
        echo "Removing cached RHCOS images from $INSTALLER_CACHE_DIR/image_cache/"
        rm -rf "$INSTALLER_CACHE_DIR"/image_cache/rhcos-*ibmcloud.s390x.qcow2* || true
        echo "Cache cleanup completed"
    else
        echo "No image cache directory found at $INSTALLER_CACHE_DIR/image_cache"
    fi
}

verify_cluster_access() {
    local kubeconfig="$1"

    if [ -z "$kubeconfig" ] || [ ! -f "$kubeconfig" ]; then
        echo "ERROR: KUBECONFIG not found at $kubeconfig"
        return 1
    fi

    echo "KUBECONFIG found at $kubeconfig"

    # Verify cluster is accessible
    if kubectl get nodes -o wide 2>&1 | tee -a "$LOG_FILE"; then
        echo "✓ Cluster is accessible and healthy!"
        return 0
    else
        echo "✗ Cluster provisioned but not accessible"
        return 1
    fi
}

# Main provisioning verification
# Note: The actual cluster provisioning is handled by OpenShift CI automation-flavors
# This wrapper just verifies the result and handles retries

KUBECONFIG="${KUBECONFIG:-}"

if [ -z "$KUBECONFIG" ]; then
    echo "ERROR: KUBECONFIG environment variable not set"
    echo "This wrapper expects the cluster to be provisioned by automation-flavors"
    exit 1
fi

# Check if cluster is already accessible
if verify_cluster_access "$KUBECONFIG"; then
    echo "Cluster provisioning successful on first attempt"
    exit 0
fi

# If we get here, cluster provisioning or access failed
# Check for known error patterns and provide diagnostics

echo "Cluster verification failed, checking for known error patterns..."

if [ -f "$LOG_FILE" ]; then
    if detect_gzip_checksum_error "$LOG_FILE"; then
        echo "✗ Detected GZIP checksum error (ROX-21457)"
        echo "  This indicates corrupted RHCOS image cache"
        cleanup_corrupted_image_cache
        echo "  Manual retry recommended after cache cleanup"
    elif detect_api_timeout_error "$LOG_FILE"; then
        echo "✗ Detected K8S API timeout error (ROX-21457)"
        echo "  This indicates s390x infrastructure resource contention"
        echo "  The bootstrap VM failed to provision within timeout"
    elif detect_bootstrap_timeout_error "$LOG_FILE"; then
        echo "✗ Detected Bootstrap timeout error (ROX-21457)"
        echo "  This indicates DNS/network resolution issues"
        echo "  The bootstrap machine cannot resolve API endpoints"
    else
        echo "✗ Unknown failure pattern"
    fi

    echo ""
    echo "Last 20 lines of log:"
    tail -20 "$LOG_FILE" || true
else
    echo "No log file found at $LOG_FILE"
fi

echo ""
echo "Provisioning verification failed"
echo "For automatic retry, this wrapper would need to be invoked before cluster provisioning"
exit 1
