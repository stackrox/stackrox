#!/usr/bin/env bash
# Deploy RHEL VMs on a KubeVirt/OpenShift Virtualization cluster.
# Manages automation SSH keypair, creates VMs, waits for SSH readiness,
# and adopts pre-existing VMs via password fallback.
#
# Designed to be sourced by add-vms.sh or run standalone.

set -euo pipefail

# Globals (set by caller or defaulted)
NAMESPACE="${NAMESPACE:-openshift-cnv}"
SSH_USER="${SSH_USER:-cloud-user}"
VM_OS="${VM_OS:-rhel9}"
VM_PREFIX="${VM_PREFIX:-${VM_OS}}"
NUM_VMS="${NUM_VMS:-1}"
CONTAINER_IMAGE="${CONTAINER_IMAGE:-quay.io/rhacs-eng/vm-images:${VM_OS}-dnf-primed-latest}"
AUTOMATION_SSH_SECRET="${AUTOMATION_SSH_SECRET:-acs-vm-automation-ssh}"
ARTIFACTS_DIR="${ARTIFACTS_DIR:-}"
USER_SSH_PUBLIC_KEY="${USER_SSH_PUBLIC_KEY:-}"
IMAGE_PULL_SECRET_NAME="${IMAGE_PULL_SECRET_NAME:-acs-vm-pull-secret}"

# Populated during execution
AUTOMATION_SSH_PRIVKEY=""
AUTOMATION_SSH_PUBKEY=""

# Arrays tracking VM state
declare -a MANAGED_VMS=()
declare -a ADOPTED_VMS=()
declare -a SKIPPED_VMS=()

die() { echo "ERROR: $*" >&2; exit 1; }

# --- SSH keypair management ---

# Loads or creates the automation SSH keypair used for all VM access.
# If the k8s secret already exists, downloads the key from it.
# Otherwise generates a new ed25519 pair and stores it in the secret.
# Sets globals: AUTOMATION_SSH_PRIVKEY, AUTOMATION_SSH_PUBKEY (temp file paths).
ensure_automation_ssh_key() {
    echo "=== Ensuring automation SSH keypair ==="

    if kubectl get secret "$AUTOMATION_SSH_SECRET" -n "$NAMESPACE" &>/dev/null; then
        echo "Loading existing automation SSH key from secret '$AUTOMATION_SSH_SECRET'..."
        AUTOMATION_SSH_PRIVKEY="$(mktemp)"
        kubectl get secret "$AUTOMATION_SSH_SECRET" -n "$NAMESPACE" \
            -o jsonpath='{.data.id_ed25519}' | base64 -d > "$AUTOMATION_SSH_PRIVKEY"
        chmod 600 "$AUTOMATION_SSH_PRIVKEY"
        AUTOMATION_SSH_PUBKEY="$(mktemp)"
        kubectl get secret "$AUTOMATION_SSH_SECRET" -n "$NAMESPACE" \
            -o jsonpath='{.data.id_ed25519\.pub}' | base64 -d > "$AUTOMATION_SSH_PUBKEY"
        echo "Loaded automation SSH key."
    else
        echo "Generating new automation SSH keypair..."
        AUTOMATION_SSH_PRIVKEY="$(mktemp)"
        rm -f "$AUTOMATION_SSH_PRIVKEY"
        AUTOMATION_SSH_PUBKEY="${AUTOMATION_SSH_PRIVKEY}.pub"
        ssh-keygen -t ed25519 -f "$AUTOMATION_SSH_PRIVKEY" -N "" -C "acs-vm-automation" -q
        echo "Storing automation SSH key in secret '$AUTOMATION_SSH_SECRET'..."
        kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
        kubectl create secret generic "$AUTOMATION_SSH_SECRET" -n "$NAMESPACE" \
            --from-file=id_ed25519="$AUTOMATION_SSH_PRIVKEY" \
            --from-file=id_ed25519.pub="$AUTOMATION_SSH_PUBKEY"
        echo "Automation SSH key created and stored."
    fi
}

# --- Image pull secret ---

ensure_image_pull_secret() {
    echo "=== Ensuring image pull secret ==="

    if [[ -z "${QUAY_RHACS_ENG_RO_USERNAME:-}" || -z "${QUAY_RHACS_ENG_RO_PASSWORD:-}" ]]; then
        die "QUAY_RHACS_ENG_RO_USERNAME and QUAY_RHACS_ENG_RO_PASSWORD must be set to create VM image pull secret"
    fi

    echo "Creating image pull secret '$IMAGE_PULL_SECRET_NAME' in namespace '$NAMESPACE'..."
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret docker-registry "$IMAGE_PULL_SECRET_NAME" \
        -n "$NAMESPACE" \
        --docker-server=quay.io \
        --docker-username="$QUAY_RHACS_ENG_RO_USERNAME" \
        --docker-password="$QUAY_RHACS_ENG_RO_PASSWORD" \
        --dry-run=client -o yaml | kubectl apply -f -
    echo "Image pull secret created/updated."
}

# --- VM creation ---

vm_exists() {
    kubectl get vm "$1" -n "$NAMESPACE" &>/dev/null
}

get_vm_status() {
    kubectl get vm "$1" -n "$NAMESPACE" -o jsonpath='{.status.printableStatus}' 2>/dev/null || echo "Unknown"
}

# Prints a KubeVirt VirtualMachine YAML manifest to stdout.
# Embeds the automation (and optional user) SSH public keys via cloud-init
# so the VM is SSH-accessible immediately after boot.
build_vm_manifest() {
    local vm_name="$1"
    local ssh_keys_yaml=""

    # Always include automation public key
    local auto_pub
    auto_pub="$(cat "$AUTOMATION_SSH_PUBKEY")"
    ssh_keys_yaml="              - ${auto_pub}"

    # Optionally include user public key
    if [[ -n "$USER_SSH_PUBLIC_KEY" ]]; then
        ssh_keys_yaml="${ssh_keys_yaml}
              - ${USER_SSH_PUBLIC_KEY}"
    fi

    cat <<EOF
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: ${vm_name}
  namespace: ${NAMESPACE}
  labels:
    app.kubernetes.io/managed-by: add-vms-automation
spec:
  runStrategy: Always
  template:
    metadata:
      labels:
        kubevirt.io/size: small
        kubevirt.io/domain: ${vm_name}
    spec:
      domain:
        cpu:
          cores: 2
          sockets: 1
          threads: 1
        devices:
          autoattachVSOCK: true
          disks:
            - name: containerdisk
              bootOrder: 1
              disk:
                bus: virtio
            - name: cloudinitdisk
              bootOrder: 2
              disk:
                bus: virtio
          interfaces:
          - name: default
            masquerade: {}
        memory:
          guest: 4Gi
        resources:
          requests:
            memory: 4Gi
            cpu: "2"
      networks:
      - name: default
        pod: {}
      volumes:
        - name: containerdisk
          containerDisk:
            image: ${CONTAINER_IMAGE}
            imagePullSecret: ${IMAGE_PULL_SECRET_NAME}
        - name: cloudinitdisk
          cloudInitNoCloud:
            userData: |
              #cloud-config
              user: ${SSH_USER}
              ssh_pwauth: false
              ssh_authorized_keys:
${ssh_keys_yaml}
EOF
}

# Creates or restarts a single VM. Idempotent: skips if running, restarts
# if stopped, creates if absent. Returns 1 only on creation failure.
create_vm() {
    local vm_name="$1" vm_index="$2"
    echo "[${vm_index}/${NUM_VMS}] VM: $vm_name"

    if vm_exists "$vm_name"; then
        local status
        status="$(get_vm_status "$vm_name")"
        echo "  Already exists (status: $status)"
        if [[ "$status" == "Running" ]]; then
            echo "  Skipping creation."
            return 0
        elif [[ "$status" == "Stopped" ]]; then
            echo "  Starting stopped VM..."
            kubectl patch vm "$vm_name" -n "$NAMESPACE" --type merge \
                -p '{"spec":{"runStrategy":"Always"}}' &>/dev/null
            return 0
        fi
        return 0
    fi

    if build_vm_manifest "$vm_name" | kubectl apply -f - &>/dev/null; then
        echo "  Created."
    else
        echo "  FAILED to create."
        return 1
    fi
}

deploy_all_vms() {
    echo "=== Deploying VMs ==="

    for i in $(seq 1 "$NUM_VMS"); do
        if ! create_vm "${VM_PREFIX}-${i}" "$i"; then
            SKIPPED_VMS+=("${VM_PREFIX}-${i}")
        fi
    done
}

# --- Wait for VMI + SSH ---

# Polls until the KubeVirt VirtualMachineInstance reports condition=Ready.
# Timeout: ~10 min (20 retries * 30s). Returns 1 on timeout.
#
# Each iteration uses a short fixed kubectl wait timeout (30s) instead of
# a dynamic one, so we get regular progress updates and can print
# diagnostics between attempts. The previous dynamic timeout
# ((max-i)*10s) caused individual kubectl wait calls to block for
# minutes, hiding the actual VM state from the CI log.
wait_for_vmi_ready() {
    local vm_name="$1" max_retries=20 i=0
    echo "  Waiting for VMI $vm_name to be ready..."
    while (( i < max_retries )); do
        if kubectl get "vmi/$vm_name" -n "$NAMESPACE" &>/dev/null; then
            if kubectl wait --for=condition=Ready "vmi/$vm_name" -n "$NAMESPACE" \
                    --timeout=30s &>/dev/null; then
                echo "  VMI $vm_name is ready."
                return 0
            fi
        fi
        i=$((i + 1))
        if (( i % 3 == 0 )); then
            echo "    Still waiting for VMI... (${i}/${max_retries})"
            dump_vmi_status "$vm_name"
        fi
    done
    echo "  VMI $vm_name did not become ready in time."
    dump_vmi_status "$vm_name"
    return 1
}

# Prints a short diagnostic for a VMI that isn't ready yet.
# Shows the VMI status/conditions and recent pod events to help
# identify scheduling failures, image pull errors, etc.
dump_vmi_status() {
    local vm_name="$1"
    echo "    --- VMI diagnostic for $vm_name ---"
    kubectl get "vmi/$vm_name" -n "$NAMESPACE" \
        -o jsonpath='    Status: {.status.phase}{"\n"}' 2>/dev/null || true
    kubectl get "vmi/$vm_name" -n "$NAMESPACE" \
        -o jsonpath='    Conditions: {.status.conditions[*].type}={.status.conditions[*].status}{"\n"}' 2>/dev/null || true
    # Show recent events for the launcher pod — surfaces image pull
    # errors, scheduling failures, and other infrastructure issues.
    local pod
    pod="$(kubectl get pods -n "$NAMESPACE" -l "kubevirt.io/domain=${vm_name}" \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
    if [[ -n "$pod" ]]; then
        echo "    Pod: $pod"
        kubectl get events -n "$NAMESPACE" --field-selector "involvedObject.name=$pod" \
            --sort-by='.lastTimestamp' 2>/dev/null | tail -5 | sed 's/^/    /' || true
    else
        echo "    No launcher pod found yet."
    fi
    echo "    ---"
}

# Single non-interactive SSH attempt using the automation key.
# Returns 0 if the VM is reachable, 1 otherwise.
ssh_probe_with_key() {
    local vm_name="$1"
    virtctl ssh \
        --namespace "$NAMESPACE" \
        --identity-file "$AUTOMATION_SSH_PRIVKEY" \
        --local-ssh-opts="-o StrictHostKeyChecking=no" \
        --local-ssh-opts="-o UserKnownHostsFile=/dev/null" \
        --local-ssh-opts="-o BatchMode=yes" \
        --local-ssh-opts="-o ConnectTimeout=10" \
        --command "echo SSH_PROBE_OK" \
        "${SSH_USER}@vmi/${vm_name}" 2>/dev/null | grep -q "SSH_PROBE_OK"
}

# Extracts the VM password from the infractl artifacts directory.
# Pre-existing infra clusters ship a vm-access.md with a password that
# we can use as a fallback when the automation key is not yet injected.
# Prints the password to stdout; returns 1 if unavailable.
parse_vm_password() {
    if [[ -z "$ARTIFACTS_DIR" ]]; then
        return 1
    fi
    local vm_access_file="${ARTIFACTS_DIR}/vm-access.md"
    if [[ ! -f "$vm_access_file" ]]; then
        return 1
    fi
    grep -oP '(?<=Password:\s).*' "$vm_access_file" 2>/dev/null | head -1
}

# "Adopts" a pre-existing VM by logging in with its password (via sshpass)
# and injecting the automation SSH public key into authorized_keys.
# After this succeeds, all further access uses key-based auth.
# The public key is base64-encoded in transit to avoid shell injection.
adopt_vm_with_password() {
    local vm_name="$1"
    local password
    password="$(parse_vm_password)" || return 1
    if [[ -z "$password" ]]; then
        return 1
    fi

    echo "  Attempting password-based adoption for $vm_name..."
    if ! command -v sshpass &>/dev/null; then
        echo "  WARNING: sshpass not installed — cannot adopt pre-existing VM."
        return 1
    fi

    local auto_pub
    auto_pub="$(cat "$AUTOMATION_SSH_PUBKEY")"

    # Use sshpass to inject automation public key (base64-encode to avoid shell injection)
    local auto_pub_b64
    auto_pub_b64="$(echo "$auto_pub" | base64 -w0)"
    if sshpass -p "$password" virtctl ssh \
        --namespace "$NAMESPACE" \
        --local-ssh-opts="-o StrictHostKeyChecking=no" \
        --local-ssh-opts="-o UserKnownHostsFile=/dev/null" \
        --local-ssh-opts="-o ConnectTimeout=10" \
        --local-ssh-opts="-o PubkeyAuthentication=no" \
        --command "mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo ${auto_pub_b64} | base64 -d >> ~/.ssh/authorized_keys && sort -u -o ~/.ssh/authorized_keys ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && echo ADOPT_OK" \
        "${SSH_USER}@vmi/${vm_name}" 2>/dev/null | grep -q "ADOPT_OK"; then
        echo "  Adopted $vm_name — automation key injected."
        return 0
    fi
    return 1
}

# Appends the caller's personal SSH public key (if provided) to the VM's
# authorized_keys. This is a convenience for developer SSH access and is
# separate from the automation key used by CI.
ensure_user_key_on_vm() {
    local vm_name="$1"
    if [[ -z "$USER_SSH_PUBLIC_KEY" ]]; then
        return 0
    fi
    echo "  Ensuring user SSH key on $vm_name..."
    local user_key_b64
    user_key_b64="$(echo "$USER_SSH_PUBLIC_KEY" | base64 -w0)"
    virtctl ssh \
        --namespace "$NAMESPACE" \
        --identity-file "$AUTOMATION_SSH_PRIVKEY" \
        --local-ssh-opts="-o StrictHostKeyChecking=no" \
        --local-ssh-opts="-o UserKnownHostsFile=/dev/null" \
        --local-ssh-opts="-o ConnectTimeout=10" \
        --command "mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo ${user_key_b64} | base64 -d >> ~/.ssh/authorized_keys && sort -u -o ~/.ssh/authorized_keys ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys" \
        "${SSH_USER}@vmi/${vm_name}" 2>/dev/null || true
}

# Retry loop (~15 min) that waits for SSH access on a VM.
# Tries key-based auth first; every 10 attempts falls back to password
# adoption for pre-existing VMs. Classifies the VM into one of:
#   MANAGED_VMS  — automation key worked (newly created VM)
#   ADOPTED_VMS  — password fallback succeeded, key now injected
#   SKIPPED_VMS  — all attempts failed, VM is inaccessible
wait_for_ssh_and_adopt() {
    local vm_name="$1"
    local max_retries=90 i=0

    echo "  Waiting for SSH on $vm_name..."
    while (( i < max_retries )); do
        if ssh_probe_with_key "$vm_name"; then
            echo "  SSH ready (automation key) on $vm_name."
            MANAGED_VMS+=("$vm_name")
            ensure_user_key_on_vm "$vm_name"
            return 0
        fi
        i=$((i + 1))

        # Periodically try password-based adoption for pre-existing VMs
        if (( i % 10 == 0 )); then
            if adopt_vm_with_password "$vm_name"; then
                ADOPTED_VMS+=("$vm_name")
                ensure_user_key_on_vm "$vm_name"
                return 0
            fi
        fi

        (( i % 5 == 0 )) && echo "    Still waiting for SSH... (${i}/${max_retries})"
        sleep 10
    done

    # Final password attempt
    if adopt_vm_with_password "$vm_name"; then
        ADOPTED_VMS+=("$vm_name")
        ensure_user_key_on_vm "$vm_name"
        return 0
    fi

    echo "  WARNING: Cannot access $vm_name — skipping."
    echo "  To grant access manually, add this public key to ~/.ssh/authorized_keys on the VM:"
    echo "    $(cat "$AUTOMATION_SSH_PUBKEY")"
    SKIPPED_VMS+=("$vm_name")
    return 1
}

vm_is_skipped() {
    local name="$1"
    local vm
    for vm in "${SKIPPED_VMS[@]+"${SKIPPED_VMS[@]}"}"; do
        [[ "$vm" == "$name" ]] && return 0
    done
    return 1
}

wait_for_all_vms() {
    echo "=== Waiting for VMs ==="
    for i in $(seq 1 "$NUM_VMS"); do
        local vm_name="${VM_PREFIX}-${i}"
        if vm_is_skipped "$vm_name"; then
            echo "  Skipping $vm_name (creation failed)."
            continue
        fi
        if ! wait_for_vmi_ready "$vm_name"; then
            SKIPPED_VMS+=("$vm_name")
            continue
        fi
        wait_for_ssh_and_adopt "$vm_name" || SKIPPED_VMS+=("$vm_name")
    done
}

# --- Entrypoint when run standalone ---

deploy_vms() {
    ensure_automation_ssh_key
    ensure_image_pull_secret
    deploy_all_vms
    wait_for_all_vms
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    deploy_vms
fi
