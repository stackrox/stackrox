//go:build test_e2e

package tests

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

var kubeVirtInstallNamespaces = []string{"openshift-cnv", "kubevirt", "openshift-kubevirt"}

const (
	kvmCapacityResourceName    = coreV1.ResourceName("devices.kubevirt.io/kvm")
	workerNodeLabel            = "node-role.kubernetes.io/worker"
	workerNodeScope            = "worker-labeled nodes"
	kvmAllSchedulableNodeScope = "all schedulable nodes"
	kvmFallbackDiagnostic      = "No worker-labeled nodes found; checking all schedulable nodes for KVM capacity"
)

type kvmPreflightNode struct {
	Name          string
	Unschedulable bool
	KVMCapacity   string
	Eligible      bool
}

type clusterKVMPreflightResult struct {
	Scope                           string
	UsedAllSchedulableNodesFallback bool
	CheckedNodes                    []kvmPreflightNode
}

func (r clusterKVMPreflightResult) IsUsable() bool {
	for _, node := range r.CheckedNodes {
		if node.Eligible {
			return true
		}
	}
	return false
}

func (r clusterKVMPreflightResult) Diagnostic() string {
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

func inspectClusterKVMReadiness(ctx context.Context, k8s kubernetes.Interface) (clusterKVMPreflightResult, error) {
	nodeList, err := k8s.CoreV1().Nodes().List(ctx, metaV1.ListOptions{})
	if err != nil {
		return clusterKVMPreflightResult{}, err
	}

	result := clusterKVMPreflightResult{
		Scope:        workerNodeScope,
		CheckedNodes: make([]kvmPreflightNode, 0, len(nodeList.Items)),
	}

	for _, node := range nodeList.Items {
		if isWorkerNode(node) {
			result.CheckedNodes = append(result.CheckedNodes, kvmPreflightNode{
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
			result.CheckedNodes = append(result.CheckedNodes, kvmPreflightNode{
				Name:          node.Name,
				Unschedulable: node.Spec.Unschedulable,
				KVMCapacity:   nodeKVMCapacityString(node),
				Eligible:      !node.Spec.Unschedulable && nodeHasPositiveKVMCapacity(node),
			})
		}
	}

	return result, nil
}

type kubeVirtVSOCKRef struct {
	Namespace string
	Name      string
}

func verifyClusterVSOCKReadyPhases(
	ctx context.Context,
	featureGateTimeout time.Duration,
	virtHandlerTimeout time.Duration,
	waitForFeatureGate func(context.Context) (kubeVirtVSOCKRef, error),
	waitForVirtHandler func(context.Context) (string, error),
) (kubeVirtVSOCKRef, string, error) {
	phaseOneCtx, cancelPhaseOne := context.WithTimeout(ctx, featureGateTimeout)
	ref, err := waitForFeatureGate(phaseOneCtx)
	cancelPhaseOne()
	if err != nil {
		return kubeVirtVSOCKRef{}, "", err
	}

	phaseTwoCtx, cancelPhaseTwo := context.WithTimeout(ctx, virtHandlerTimeout)
	lastDiag, err := waitForVirtHandler(phaseTwoCtx)
	cancelPhaseTwo()
	if err != nil {
		return ref, lastDiag, err
	}

	return ref, lastDiag, nil
}

func virtHandlerHostVsockVolumesLookUsable(ctx context.Context, t testing.TB, k8s kubernetes.Interface) (bool, string) {
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
			phaseReady := pod.Status.Phase == coreV1.PodRunning || pod.Status.Phase == coreV1.PodPending
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
				// OCP Virtualization commonly exposes vsock plumbing through libvirt runtime mounts
				// without an explicit "/dev/vhost-vsock" hostPath in the pod spec.
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
		// Fprintf to a Builder avoids the intermediate string allocation of Sprintf+WriteString.
		fmt.Fprintf(&b, "%s->%q", vol.Name, vol.HostPath.Path)
		n++
	}
	if n == 0 {
		return "<none>"
	}
	return b.String()
}
