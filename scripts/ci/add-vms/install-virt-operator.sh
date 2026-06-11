#!/usr/bin/env bash
# Install OpenShift Virtualization (HyperConverged) operator.
# Idempotent: skips if HCO is already healthy.

set -euo pipefail

OLM_NAMESPACE="openshift-cnv"
HCO_NAME="kubevirt-hyperconverged"
SUBSCRIPTION_NAME="kubevirt-hyperconverged"

die() { echo "ERROR: $*" >&2; exit 1; }

hco_condition() {
    local cond="$1"
    kubectl -n "$OLM_NAMESPACE" get hyperconverged "$HCO_NAME" \
        -o jsonpath="{.status.conditions[?(@.type==\"${cond}\")].status}" 2>/dev/null || echo "Unknown"
}

hco_is_healthy() {
    local avail prog degr
    avail="$(hco_condition Available)"
    prog="$(hco_condition Progressing)"
    degr="$(hco_condition Degraded)"
    [[ "$avail" == "True" && "$prog" == "False" && "$degr" == "False" ]]
}

wait_for_virt_handler_rollout() {
    echo "Waiting for virt-handler rollout..."
    kubectl rollout status ds/virt-handler -n "$OLM_NAMESPACE" --timeout=600s \
        || die "virt-handler did not roll out successfully"
}

install_virt_operator() {
    echo "=== OpenShift Virtualization Installer ==="

    command -v kubectl &>/dev/null || die "kubectl is not installed"
    kubectl cluster-info &>/dev/null || die "Cannot connect to Kubernetes cluster"

    # Idempotent: skip if already healthy
    if kubectl get hyperconverged "$HCO_NAME" -n "$OLM_NAMESPACE" &>/dev/null && hco_is_healthy; then
        echo "OpenShift Virtualization is already installed and healthy — skipping."
        return 0
    fi

    echo "Installing namespace, OperatorGroup, and Subscription..."
    kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Namespace
metadata:
  name: openshift-cnv
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: openshift-cnv
  namespace: openshift-cnv
spec:
  targetNamespaces:
  - openshift-cnv
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: kubevirt-hyperconverged
  namespace: openshift-cnv
spec:
  channel: stable
  name: kubevirt-hyperconverged
  source: redhat-operators
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic
EOF

    # Wait for installedCSV (up to 5 min)
    echo "Waiting for Subscription to report installedCSV..."
    local elapsed=0
    until kubectl -n "$OLM_NAMESPACE" get sub "$SUBSCRIPTION_NAME" \
            -o jsonpath='{.status.installedCSV}' 2>/dev/null | grep -q .; do
        sleep 5; elapsed=$((elapsed + 5))
        if (( elapsed >= 300 )); then
            die "Timeout waiting for installedCSV after ${elapsed}s"
        fi
        (( elapsed % 30 == 0 )) && echo "  Still waiting... (${elapsed}s)"
    done

    local csv
    csv="$(kubectl -n "$OLM_NAMESPACE" get sub "$SUBSCRIPTION_NAME" -o jsonpath='{.status.installedCSV}')"
    echo "InstalledCSV: $csv"

    # Wait for CSV to reach Succeeded (up to 15 min)
    echo "Waiting for CSV to reach Succeeded..."
    elapsed=0
    while true; do
        local phase
        phase="$(kubectl -n "$OLM_NAMESPACE" get csv "$csv" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
        if [[ "$phase" == "Succeeded" ]]; then
            echo "CSV is Succeeded"
            break
        fi
        sleep 5; elapsed=$((elapsed + 5))
        if (( elapsed >= 900 )); then
            die "CSV did not reach Succeeded after ${elapsed}s (current: ${phase:-Unknown})"
        fi
        (( elapsed % 60 == 0 )) && echo "  CSV phase: ${phase:-Unknown} (${elapsed}s)"
    done

    # Create HyperConverged CR (without VSOCK initially — added separately if needed)
    echo "Creating HyperConverged CR..."
    kubectl apply -f - <<EOF
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: ${HCO_NAME}
  namespace: ${OLM_NAMESPACE}
spec: {}
EOF

    # Wait for HCO healthy (up to 30 min)
    echo "Waiting for HyperConverged to become healthy..."
    elapsed=0
    while true; do
        if hco_is_healthy; then
            echo "HyperConverged is healthy"
            break
        fi
        sleep 10; elapsed=$((elapsed + 10))
        if (( elapsed >= 1800 )); then
            die "Timeout waiting for HCO to become healthy after ${elapsed}s"
        fi
        if (( elapsed % 60 == 0 )); then
            echo "  Status: Available=$(hco_condition Available), Progressing=$(hco_condition Progressing), Degraded=$(hco_condition Degraded) (${elapsed}s)"
        fi
    done

    # Enable VSOCK feature gate if not already present (check KubeVirt CR, patch via HCO annotation)
    local kv_gates
    kv_gates="$(kubectl get kubevirt -n "$OLM_NAMESPACE" \
        -o jsonpath='{.items[0].spec.configuration.developerConfiguration.featureGates}' 2>/dev/null || true)"
    if [[ "$kv_gates" != *"VSOCK"* ]]; then
        echo "Annotating HyperConverged CR to add VSOCK feature gate..."
        local vsock_patch='[{"op":"add","path":"/spec/configuration/developerConfiguration/featureGates/-","value":"VSOCK"}]'
        kubectl annotate hyperconverged "$HCO_NAME" -n "$OLM_NAMESPACE" --overwrite \
            "kubevirt.kubevirt.io/jsonpatch=${vsock_patch}"

        echo "Waiting for VSOCK to appear in KubeVirt CR feature gates..."
        local vsock_elapsed=0
        while (( vsock_elapsed < 300 )); do
            kv_gates="$(kubectl get kubevirt -n "$OLM_NAMESPACE" \
                -o jsonpath='{.items[0].spec.configuration.developerConfiguration.featureGates}' 2>/dev/null || true)"
            if [[ "$kv_gates" == *"VSOCK"* ]]; then
                echo "VSOCK feature gate is active."
                break
            fi
            sleep 5; vsock_elapsed=$((vsock_elapsed + 5))
            (( vsock_elapsed % 30 == 0 )) && echo "  Still waiting for VSOCK... (${vsock_elapsed}s)"
        done
        if [[ "$kv_gates" != *"VSOCK"* ]]; then
            die "KubeVirt CR still missing VSOCK after ${vsock_elapsed}s"
        fi
    else
        echo "VSOCK feature gate already enabled."
    fi
    wait_for_virt_handler_rollout

    # Patch subscription with KVM_EMULATION
    local current_kvm
    current_kvm="$(kubectl get subscription "$SUBSCRIPTION_NAME" -n "$OLM_NAMESPACE" \
        -o jsonpath='{.spec.config.env[?(@.name=="KVM_EMULATION")].value}' 2>/dev/null || echo "")"
    if [[ "$current_kvm" != "true" ]]; then
        echo "Patching subscription with KVM_EMULATION=true..."
        # Ensure spec.config.env exists, then add the entry without clobbering other env vars
        kubectl get subscription "$SUBSCRIPTION_NAME" -n "$OLM_NAMESPACE" -o json \
            | jq '.spec.config.env = ((.spec.config.env // []) | map(select(.name != "KVM_EMULATION")) + [{"name":"KVM_EMULATION","value":"true"}])
                 | .spec.config.selector = {"matchLabels":{"name":"hyperconverged-cluster-operator"}}' \
            | kubectl apply -f -
    else
        echo "KVM_EMULATION already set."
    fi

    echo "=== OpenShift Virtualization installed successfully ==="
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    install_virt_operator
fi
