package vmhelpers

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	kvmCapacityResourceName = coreV1.ResourceName("devices.kubevirt.io/kvm")
	workerNodeLabel         = "node-role.kubernetes.io/worker"

	workerNodeScope            = "worker-labeled nodes"
	kvmAllSchedulableNodeScope = "all schedulable nodes"
	kvmFallbackDiagnostic      = "No worker-labeled nodes found; checking all schedulable nodes for KVM capacity"
)

// KVMPreflightNode summarizes whether a node can host KubeVirt VMs.
type KVMPreflightNode struct {
	Name          string
	Unschedulable bool
	KVMCapacity   string
	Eligible      bool
}

// ClusterKVMPreflightResult describes the nodes checked during KVM preflight.
type ClusterKVMPreflightResult struct {
	Scope                           string
	UsedAllSchedulableNodesFallback bool
	CheckedNodes                    []KVMPreflightNode
}

// IsUsable reports whether at least one checked node can host a VM.
func (r ClusterKVMPreflightResult) IsUsable() bool {
	for _, node := range r.CheckedNodes {
		if node.Eligible {
			return true
		}
	}
	return false
}

// Diagnostic formats a human-readable KVM preflight summary.
func (r ClusterKVMPreflightResult) Diagnostic() string {
	parts := make([]string, 0, 2)
	if r.UsedAllSchedulableNodesFallback {
		parts = append(parts, kvmFallbackDiagnostic)
	}

	if r.IsUsable() {
		eligibleNodes := make([]string, 0, len(r.CheckedNodes))
		for _, node := range r.CheckedNodes {
			if node.Eligible {
				eligibleNodes = append(eligibleNodes, fmt.Sprintf("%s(kvm=%s)", node.Name, node.KVMCapacity))
			}
		}
		parts = append(parts, fmt.Sprintf("KVM-capable %s: %s", r.Scope, strings.Join(eligibleNodes, " ")))
		return strings.Join(parts, "\n")
	}

	checkedNodes := make([]string, 0, len(r.CheckedNodes))
	for _, node := range r.CheckedNodes {
		checkedNodes = append(checkedNodes, fmt.Sprintf("%s(unschedulable=%t kvm=%s)", node.Name, node.Unschedulable, node.KVMCapacity))
	}
	parts = append(parts, fmt.Sprintf("No %s with devices.kubevirt.io/kvm > 0. Checked: %s", r.Scope, strings.Join(checkedNodes, " ")))
	return strings.Join(parts, "\n")
}

func isWorkerNode(node coreV1.Node) bool {
	_, ok := node.Labels[workerNodeLabel]
	return ok
}

func nodeHasPositiveKVMCapacity(node coreV1.Node) bool {
	quantity, ok := node.Status.Capacity[kvmCapacityResourceName]
	return ok && quantity.Sign() > 0
}

func nodeKVMCapacityString(node coreV1.Node) string {
	quantity, ok := node.Status.Capacity[kvmCapacityResourceName]
	if !ok {
		return "<unset>"
	}
	return quantity.String()
}

// InspectClusterKVMReadiness inspects cluster nodes and reports whether any are suitable for KubeVirt VMs.
func InspectClusterKVMReadiness(ctx context.Context, k8s kubernetes.Interface) (ClusterKVMPreflightResult, error) {
	nodeList, err := k8s.CoreV1().Nodes().List(ctx, metaV1.ListOptions{})
	if err != nil {
		return ClusterKVMPreflightResult{}, err
	}

	result := ClusterKVMPreflightResult{
		Scope:        workerNodeScope,
		CheckedNodes: make([]KVMPreflightNode, 0, len(nodeList.Items)),
	}

	for _, node := range nodeList.Items {
		if isWorkerNode(node) {
			result.CheckedNodes = append(result.CheckedNodes, KVMPreflightNode{
				Name:          node.Name,
				Unschedulable: node.Spec.Unschedulable,
				KVMCapacity:   nodeKVMCapacityString(node),
				Eligible:      !node.Spec.Unschedulable && nodeHasPositiveKVMCapacity(node),
			})
		}
	}

	if len(result.CheckedNodes) == 0 {
		result.Scope = kvmAllSchedulableNodeScope
		result.UsedAllSchedulableNodesFallback = true
		for _, node := range nodeList.Items {
			if node.Spec.Unschedulable {
				continue
			}
			result.CheckedNodes = append(result.CheckedNodes, KVMPreflightNode{
				Name:          node.Name,
				Unschedulable: node.Spec.Unschedulable,
				KVMCapacity:   nodeKVMCapacityString(node),
				Eligible:      !node.Spec.Unschedulable && nodeHasPositiveKVMCapacity(node),
			})
		}
	}

	return result, nil
}

// KubeVirtVSOCKRef identifies the KubeVirt custom resource that enables the VSOCK feature gate.
type KubeVirtVSOCKRef struct {
	Namespace string
	Name      string
}

// VerifyClusterVSOCKReadyPhases runs the feature-gate and virt-handler checks with separate timeout budgets.
func VerifyClusterVSOCKReadyPhases(
	ctx context.Context,
	featureGateTimeout time.Duration,
	virtHandlerTimeout time.Duration,
	waitForFeatureGate func(context.Context) (KubeVirtVSOCKRef, error),
	waitForVirtHandler func(context.Context) (string, error),
) (KubeVirtVSOCKRef, string, error) {
	phaseOneCtx, cancelPhaseOne := context.WithTimeout(ctx, featureGateTimeout)
	defer cancelPhaseOne()
	ref, err := waitForFeatureGate(phaseOneCtx)
	if err != nil {
		return KubeVirtVSOCKRef{}, "", err
	}

	phaseTwoCtx, cancelPhaseTwo := context.WithTimeout(ctx, virtHandlerTimeout)
	defer cancelPhaseTwo()
	lastDiag, err := waitForVirtHandler(phaseTwoCtx)
	if err != nil {
		return ref, lastDiag, err
	}

	return ref, lastDiag, nil
}

// VirtHandlerHostVsockVolumesLookUsable checks whether virt-handler pods expose enough host evidence for vsock plumbing.
func VirtHandlerHostVsockVolumesLookUsable(ctx context.Context, t testing.TB, k8s kubernetes.Interface, kubeVirtInstallNamespaces ...string) (bool, string) {
	t.Helper()
	var diag strings.Builder
	selectors := []string{
		"kubevirt.io=virt-handler",
		"app.kubernetes.io/component=virt-handler",
	}
	for _, ns := range kubeVirtInstallNamespaces {
		var pods *coreV1.PodList
		foundPods := false
		for _, sel := range selectors {
			list, lerr := k8s.CoreV1().Pods(ns).List(ctx, metaV1.ListOptions{LabelSelector: sel})
			if lerr != nil {
				fmt.Fprintf(&diag, "namespace %q: list pods (%q): %v\n", ns, sel, lerr)
				continue
			}
			if len(list.Items) > 0 {
				pods = list
				foundPods = true
				break
			}
		}
		if !foundPods {
			fmt.Fprintf(&diag, "namespace %q: no virt-handler pods for selectors %v\n", ns, selectors)
			continue
		}
		for i := range pods.Items {
			pod := &pods.Items[i]
			// Waiting for PodRunning, as PodPending pods may declare hostPath volumes that aren't mounted yet.
			phaseReady := pod.Status.Phase == coreV1.PodRunning
			if !phaseReady {
				fmt.Fprintf(&diag, "namespace %q pod %q: phase=%q\n", ns, pod.Name, pod.Status.Phase)
			}
			hasExplicitVsockPath := false
			hasCNVLibvirtRuntimePath := false
			for _, vol := range pod.Spec.Volumes {
				if vol.HostPath == nil {
					continue
				}
				p := strings.ToLower(vol.HostPath.Path)
				if strings.Contains(p, "vsock") {
					hasExplicitVsockPath = true
					break
				}
				if strings.Contains(p, "kubevirt-libvirt-runtimes") {
					hasCNVLibvirtRuntimePath = true
				}
			}
			if phaseReady {
				if hasExplicitVsockPath {
					return true, diag.String()
				}
				if hasCNVLibvirtRuntimePath {
					fmt.Fprintf(&diag, "namespace %q pod %q: accepting CNV libvirt runtime hostPath as vsock evidence\n", ns, pod.Name)
					return true, diag.String()
				}
			}
			fmt.Fprintf(&diag, "namespace %q pod %q: hostPath volumes (no vsock-like path): %s\n",
				ns, pod.Name, summarizePodHostPathVolumes(pod))
		}
	}
	return false, diag.String()
}

func summarizePodHostPathVolumes(pod *coreV1.Pod) string {
	var b strings.Builder
	n := 0
	for _, vol := range pod.Spec.Volumes {
		if vol.HostPath == nil {
			continue
		}
		if n > 0 {
			b.WriteString("; ")
		}
		fmt.Fprintf(&b, "%s->%q", vol.Name, vol.HostPath.Path)
		n++
	}
	if n == 0 {
		return "<none>"
	}
	return b.String()
}
