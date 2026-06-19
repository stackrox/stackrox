//go:build test_e2e_vm

package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/tests/vmhelpers"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var kubeVirtInstallNamespaces = []string{"openshift-cnv", "kubevirt", "openshift-kubevirt"}

const (
	vsockFeatureGateWaitTimeout         = 3 * time.Minute
	vsockVirtHandlerEvidenceWaitTimeout = 3 * time.Minute
	vsockPreflightPollInterval          = 5 * time.Second
)

func mustVerifyClusterKVMReady(t testing.TB, ctx context.Context, k8s kubernetes.Interface) {
	t.Helper()
	result, err := vmhelpers.InspectClusterKVMReadiness(ctx, k8s)
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

func waitForKubeVirtWithVSOCKFeatureGate(ctx context.Context, t testing.TB, dyn dynamic.Interface) (vmhelpers.KubeVirtVSOCKRef, error) {
	t.Helper()
	var ref vmhelpers.KubeVirtVSOCKRef
	err := wait.PollUntilContextCancel(ctx, vsockPreflightPollInterval, true, func(ctx context.Context) (bool, error) {
		ns, kvObj, _, err := findKubeVirtWithVSOCKFeatureGate(ctx, dyn)
		if err != nil {
			t.Logf("VSOCK preflight: KubeVirt CR with VSOCK feature gate not found yet: %v", err)
			return false, nil
		}
		ref = vmhelpers.KubeVirtVSOCKRef{
			Namespace: ns,
			Name:      kvObj.GetName(),
		}
		return true, nil
	})
	if err != nil {
		return vmhelpers.KubeVirtVSOCKRef{}, err
	}
	return ref, nil
}

func waitForVirtHandlerVsockEvidence(ctx context.Context, t testing.TB, k8s kubernetes.Interface) (string, error) {
	t.Helper()
	var lastDiag string
	err := wait.PollUntilContextCancel(ctx, vsockPreflightPollInterval, true, func(ctx context.Context) (bool, error) {
		ok, diag := vmhelpers.VirtHandlerHostVsockVolumesLookUsable(ctx, t, k8s, kubeVirtInstallNamespaces...)
		lastDiag = diag
		return ok, nil
	})
	return lastDiag, err
}

// mustVerifyClusterVSOCKReady checks KubeVirt VSOCK enablement and host vsock plumbing used by the virt stack.
// It fails with explicit diagnostics suitable for CI triage when either check cannot be satisfied.
func mustVerifyClusterVSOCKReady(t *testing.T, ctx context.Context, k8s kubernetes.Interface, dyn dynamic.Interface) {
	t.Helper()
	ref, lastDiag, err := vmhelpers.VerifyClusterVSOCKReadyPhases(
		ctx,
		vsockFeatureGateWaitTimeout,
		vsockVirtHandlerEvidenceWaitTimeout,
		func(phaseCtx context.Context) (vmhelpers.KubeVirtVSOCKRef, error) {
			return waitForKubeVirtWithVSOCKFeatureGate(phaseCtx, t, dyn)
		},
		func(phaseCtx context.Context) (string, error) {
			return waitForVirtHandlerVsockEvidence(phaseCtx, t, k8s)
		},
	)
	if err != nil {
		if ref == (vmhelpers.KubeVirtVSOCKRef{}) {
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
