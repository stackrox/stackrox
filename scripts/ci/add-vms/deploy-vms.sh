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

wait_for_vmi_ready() {
    local vm_name="$1" max_retries=30 i=0
    echo "  Waiting for VMI $vm_name to be ready..."
    while (( i < max_retries )); do
        if kubectl get "vmi/$vm_name" -n "$NAMESPACE" &>/dev/null; then
            if kubectl wait --for=condition=Ready "vmi/$vm_name" -n "$NAMESPACE" \
                    --timeout="$(( (max_retries - i) * 10 ))s" &>/dev/null; then
                echo "  VMI $vm_name is ready."
                return 0
            fi
        fi
        i=$((i + 1))
        (( i % 5 == 0 )) && echo "    Still waiting for VMI... (${i}/${max_retries})"
        sleep 10
    done
    echo "  VMI $vm_name did not become ready in time."
    return 1
}

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
