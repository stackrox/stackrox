#!/usr/bin/env bash

set -euo pipefail

# Required to start:
# - Create a new OCP cluster
# - install virtualization operator
# - set KUBECONFIG
# - Deploy ACS
# - Enable ROX_VIRTUAL_MACHINES

# Deployment behavior (matches CI lane intent)
export ORCHESTRATOR_FLAVOR=openshift
export DEPLOY_STACKROX_VIA_OPERATOR=true
export SENSOR_SCANNER_SUPPORT=true
export ROX_DEPLOY_SENSOR_WITH_CRS=true
export SENSOR_HELM_MANAGED=true
export ROX_VIRTUAL_MACHINES=true

# VM-scanning required inputs
export VIRTCTL_PATH="$(command -v virtctl)"
export ROXAGENT_BINARY_PATH="$PWD/bin/linux_amd64/roxagent"

# From your cluster's virtualization boot sources:
# export VM_IMAGE_RHEL9="$(oc get istag -n openshift-virtualization-os-images rhel9-guest:latest -o jsonpath='{.image.dockerImageReference}')"
export VM_IMAGE_RHEL9="quay.io/prygiels/rhel9-dnf-primed:latest"
# export VM_IMAGE_RHEL10="$(oc get istag -n openshift-virtualization-os-images rhel10-guest:latest -o jsonpath='{.image.dockerImageReference}')"
export VM_IMAGE_RHEL10="quay.io/prygiels/rhel10-dnf-primed:latest"
export VM_IMAGE_PULL_SECRET_PATH="$HOME/.config/containers/auth.json"

# Guest users for RHEL cloud images:
export VM_GUEST_USER_RHEL9="cloud-user"
export VM_GUEST_USER_RHEL10="cloud-user"

export API_ENDPOINT="$(oc -n stackrox get route central -o jsonpath='{.spec.host}'):443"
export ROX_USERNAME="admin"
export ROX_ADMIN_PASSWORD="admin"

# Persistent, dedicated SSH keypair for VM scanning E2E.
# Reused across runs so manual troubleshooting can reuse the same key path.
vm_scan_ssh_key="$HOME/.ssh/id_stackrox_vm_scan_e2e_rsa"
mkdir -p "$HOME/.ssh"
chmod 700 "$HOME/.ssh"
if [[ ! -f "$vm_scan_ssh_key" ]]; then
    ssh-keygen -q -t rsa -b 4096 -N "" -f "$vm_scan_ssh_key" -C "stackrox-vm-scan-e2e" >/dev/null
fi
if [[ ! -f "${vm_scan_ssh_key}.pub" ]]; then
    ssh-keygen -y -f "$vm_scan_ssh_key" > "${vm_scan_ssh_key}.pub"
fi
chmod 600 "$vm_scan_ssh_key"
chmod 644 "${vm_scan_ssh_key}.pub"

export VM_SSH_PRIVATE_KEY="$(cat "$vm_scan_ssh_key")"          # PEM content of the private key
export VM_SSH_PUBLIC_KEY="$(cat "${vm_scan_ssh_key}.pub")"    # authorized_keys line for cloud-init


export ROXAGENT_REPO2CPE_PRIMARY_URL="https://security.access.redhat.com/data/metrics/repository-to-cpe.json"
export ROXAGENT_REPO2CPE_FALLBACK_URL="https://security.access.redhat.com/data/metrics/repository-to-cpe.json"
export ROXAGENT_REPO2CPE_PRIMARY_ATTEMPTS=3

export VM_SCAN_NAMESPACE_PREFIX="vm-scan-e2e"
# TEMPORARY local-dev override: force one namespace across manual runs.
export VM_SCAN_NAMESPACE="vm-scan-e2e-manual"
export VM_SCAN_TIMEOUT=20m
export VM_SCAN_POLL_INTERVAL=10s
export VM_SCAN_ESCALATION_ATTEMPT=5
export VM_DELETE_TIMEOUT=5m
export VM_SCAN_SKIP_CLEANUP=false   # keep VMs and namespace after test run for faster iteration

export VM_SCAN_REQUIRE_ACTIVATION=true   # or true + activation creds
export RHEL_ACTIVATION_ORG="<org>"
export RHEL_ACTIVATION_KEY="<secret>"

echo "Manual SSH (rhel9):"
echo "${VIRTCTL_PATH} ssh -n ${VM_SCAN_NAMESPACE} --identity-file \"${vm_scan_ssh_key}\" --username \"${VM_GUEST_USER_RHEL9}\" vmi/vm-rhel9"
echo
echo "Manual SSH command check (sudo):"
echo "${VIRTCTL_PATH} ssh -n ${VM_SCAN_NAMESPACE} --identity-file \"${vm_scan_ssh_key}\" --username \"${VM_GUEST_USER_RHEL9}\" vmi/vm-rhel9 --command '\"sudo\" \"-n\" \"true\"'"
echo
echo "Building roxagent..."
make roxagent_linux-amd64
echo
echo "Running vmhelpers tests..."
go test -race -p 1 -timeout 90m ./tests/vmhelpers -v
echo
echo "Running unit tests..."
go test -tags test -race -count=1 -v ./tests/testmetrics
go test -tags test_e2e ./tests -run TestLoadVMScanConfig -v
echo
echo "Running vmscanning tests..."
go test -tags test_e2e -run TestVMScanning -v -count=1 -p 1 -timeout 120m ./tests

echo "Done!"
