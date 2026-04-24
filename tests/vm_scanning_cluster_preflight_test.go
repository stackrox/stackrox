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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var kubeVirtInstallNamespaces = []string{"openshift-cnv", "kubevirt", "openshift-kubevirt"}

const (
	kvmCapacityResourceName             = coreV1.ResourceName("devices.kubevirt.io/kvm")
	kvmWorkerNodeLabel                  = "node-role.kubernetes.io/worker"
	kvmWorkerNodeScope                  = "worker-labeled nodes"
	kvmAllSchedulableNodeScope          = "all schedulable nodes"
	kvmFallbackDiagnostic               = "No worker-labeled nodes found; checking all schedulable nodes for KVM capacity"
	vsockFeatureGateWaitTimeout         = 3 * time.Minute
	vsockVirtHandlerEvidenceWaitTimeout = 3 * time.Minute
	vsockPreflightPollInterval          = 5 * time.Second
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
	_, ok := node.Labels[kvmWorkerNodeLabel]
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
		Scope:        kvmWorkerNodeScope,
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
		result.CheckedNodes = make([]kvmPreflightNode, 0, len(nodeList.Items))
		for _, node := range nodeList.Items {
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

func mustVerifyClusterKVMReady(t testing.TB, ctx context.Context, k8s kubernetes.Interface) {
	t.Helper()
	result, err := inspectClusterKVMReadiness(ctx, k8s)
	if err != nil {
		t.Fatalf("KVM preflight: could not inspect cluster nodes: %v", err)
	}
	if !result.IsUsable() {
		t.Fatalf("KVM preflight: no nodes suitable for KubeVirt VM scheduling.\n%s\nRemediation: ensure at least one schedulable worker advertises devices.kubevirt.io/kvm > 0 and nested virtualization is enabled.", result.Diagnostic())
	}
	t.Logf("KVM preflight: %s", result.Diagnostic())
}

var kubevirtCRGVR = schema.GroupVersionResource{
	Group:    "kubevirt.io",
	Version:  "v1",
	Resource: "kubevirts",
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

func waitForKubeVirtWithVSOCKFeatureGate(ctx context.Context, dyn dynamic.Interface) (kubeVirtVSOCKRef, error) {
	var ref kubeVirtVSOCKRef
	err := wait.PollUntilContextCancel(ctx, vsockPreflightPollInterval, true, func(ctx context.Context) (bool, error) {
		ns, kvObj, _, err := findKubeVirtWithVSOCKFeatureGate(ctx, dyn)
		if err != nil {
			return false, nil
		}
		ref = kubeVirtVSOCKRef{
			Namespace: ns,
			Name:      kvObj.GetName(),
		}
		return true, nil
	})
	if err != nil {
		return kubeVirtVSOCKRef{}, err
	}
	return ref, nil
}

func waitForVirtHandlerVsockEvidence(ctx context.Context, t testing.TB, k8s kubernetes.Interface) (string, error) {
	t.Helper()
	var lastDiag string
	err := wait.PollUntilContextCancel(ctx, vsockPreflightPollInterval, true, func(ctx context.Context) (bool, error) {
		ok, diag := virtHandlerHostVsockVolumesLookUsable(ctx, t, k8s)
		lastDiag = diag
		return ok, nil
	})
	return lastDiag, err
}

// mustVerifyClusterVSOCKReady checks KubeVirt VSOCK enablement and host vsock plumbing used by the virt stack.
// It fails with explicit diagnostics suitable for CI triage when either check cannot be satisfied.
func mustVerifyClusterVSOCKReady(t *testing.T, ctx context.Context, k8s kubernetes.Interface, dyn dynamic.Interface) {
	t.Helper()
	ref, lastDiag, err := verifyClusterVSOCKReadyPhases(
		ctx,
		vsockFeatureGateWaitTimeout,
		vsockVirtHandlerEvidenceWaitTimeout,
		func(phaseCtx context.Context) (kubeVirtVSOCKRef, error) {
			return waitForKubeVirtWithVSOCKFeatureGate(phaseCtx, dyn)
		},
		func(phaseCtx context.Context) (string, error) {
			return waitForVirtHandlerVsockEvidence(phaseCtx, t, k8s)
		},
	)
	if err != nil {
		if ref == (kubeVirtVSOCKRef{}) {
			t.Fatalf("VSOCK preflight: no KubeVirt CR with VSOCK feature gate found after %s.\n"+
				"Tried namespaces %s.\n"+
				"Remediation: install/configure KubeVirt/CNV and add \"VSOCK\" to spec.configuration.developerConfiguration.featureGates on a KubeVirt CR (see KubeVirt docs for vsock).",
				vsockFeatureGateWaitTimeout, strings.Join(kubeVirtInstallNamespaces, ", "))
		}

		t.Fatalf("VSOCK preflight: virt-handler does not show usable vsock plumbing evidence after %s.\n"+
			"KubeVirt CR %q/%q has VSOCK feature gate enabled, but virt-handler pod evidence checks failed.\n"+
			"%s\n"+
			"Remediation: ensure worker nodes provide /dev/vhost-vsock (or equivalent) and virt-handler pods expose either "+
			"an explicit vsock hostPath or the CNV libvirt runtime hostPath (/var/run/kubevirt-libvirt-runtimes); "+
			"guest roxagent requires a working virtio-vsock channel (/dev/vsock in the guest once the VMI is created).",
			vsockVirtHandlerEvidenceWaitTimeout, ref.Namespace, ref.Name, lastDiag)
	}
}

// findKubeVirtWithVSOCKFeatureGate scans every KubeVirt CR in candidate install namespaces and returns
// the first whose developerConfiguration.featureGates lists VSOCK (case-insensitive). If none qualify,
// the error message aggregates per-CR diagnostics for CI triage.
func findKubeVirtWithVSOCKFeatureGate(ctx context.Context, dyn dynamic.Interface) (ns string, obj unstructured.Unstructured, fg []string, err error) {
	var diag strings.Builder
	var lastListErr error
	for _, candidate := range kubeVirtInstallNamespaces {
		list, lerr := dyn.Resource(kubevirtCRGVR).Namespace(candidate).List(ctx, metaV1.ListOptions{})
		if lerr != nil {
			lastListErr = lerr
			fmt.Fprintf(&diag, "namespace %q: list KubeVirt CRs failed: %v\n", candidate, lerr)
			continue
		}
		if len(list.Items) == 0 {
			fmt.Fprintf(&diag, "namespace %q: no KubeVirt custom resources\n", candidate)
			continue
		}
		for _, item := range list.Items {
			key := fmt.Sprintf("%s/%s", item.GetNamespace(), item.GetName())
			fgList, found, nerr := unstructured.NestedStringSlice(item.Object, "spec", "configuration", "developerConfiguration", "featureGates")
			if nerr != nil {
				fmt.Fprintf(&diag, "KubeVirt %s: could not read featureGates: %v\n", key, nerr)
				continue
			}
			if !found {
				fmt.Fprintf(&diag, "KubeVirt %s: spec.configuration.developerConfiguration.featureGates missing\n", key)
				continue
			}
			hasVsock := false
			for _, g := range fgList {
				if strings.EqualFold(strings.TrimSpace(g), "VSOCK") {
					hasVsock = true
					break
				}
			}
			if hasVsock {
				return item.GetNamespace(), item, fgList, nil
			}
			phase, _, _ := unstructured.NestedString(item.Object, "status", "phase")
			fmt.Fprintf(&diag, "KubeVirt %s: VSOCK not in featureGates (status.phase=%q featureGates=%v)\n", key, phase, fgList)
		}
	}
	summary := strings.TrimSpace(diag.String())
	if summary == "" {
		summary = "no KubeVirt CRs discovered in candidate namespaces"
	}
	if lastListErr != nil {
		return "", unstructured.Unstructured{}, nil, fmt.Errorf("last list error: %w\n%s", lastListErr, summary)
	}
	return "", unstructured.Unstructured{}, nil, fmt.Errorf("%s", summary)
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
				if strings.Contains(p, "vhost_vsock") || strings.Contains(p, "vsock") {
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
		fmt.Fprintf(&b, "%s->%q", vol.Name, vol.HostPath.Path)
		n++
	}
	if n == 0 {
		return "<none>"
	}
	return b.String()
}
