//go:build test_e2e

package tests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func mustLoadVMScanConfig(t *testing.T) *vmScanConfig {
	t.Helper()
	cfg, err := loadVMScanConfig()
	require.NoError(t, err, "loadVMScanConfig")
	return cfg
}

func mustCreateDynamicClient(t *testing.T, restCfg *rest.Config) dynamic.Interface {
	t.Helper()
	c, err := dynamic.NewForConfig(restCfg)
	require.NoError(t, err, "dynamic.NewForConfig")
	return c
}

// mustResolveSSHIdentityFile writes the PEM-encoded private key content to a temporary file
// with 0600 permissions and returns the path, suitable for virtctl --identity-file.
func mustResolveSSHIdentityFile(t *testing.T, cfg *vmScanConfig) string {
	t.Helper()
	content := cfg.SSHPrivateKey
	require.NotEmpty(t, strings.TrimSpace(content), "SSH private key content is empty")
	require.True(t, strings.HasPrefix(strings.TrimSpace(content), "-----BEGIN"),
		"VM_SSH_PRIVATE_KEY must contain PEM-encoded key content, not a file path")

	// OpenSSH requires a trailing newline after the END marker.
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	f, err := os.CreateTemp(t.TempDir(), "vm-scan-ssh-*")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0o600))
	return f.Name()
}

func vmScanVirtualMachinesFeatureEnvVar() string {
	return features.VirtualMachines.EnvVar()
}

func formatFeatureFlagsForDiag(resp *v1.GetFeatureFlagsResponse) string {
	if resp == nil {
		return "<nil response>"
	}
	var b strings.Builder
	for _, f := range resp.GetFeatureFlags() {
		fmt.Fprintf(&b, "%s=%v; ", f.GetEnvVar(), f.GetEnabled())
	}
	return strings.TrimSpace(b.String())
}

var kubevirtCRGVR = schema.GroupVersionResource{
	Group:    "kubevirt.io",
	Version:  "v1",
	Resource: "kubevirts",
}

// mustVerifyClusterVSOCKReady checks KubeVirt VSOCK enablement and host vsock plumbing used by the virt stack.
// It fails with explicit diagnostics suitable for CI triage when either check cannot be satisfied.
func mustVerifyClusterVSOCKReady(t *testing.T, ctx context.Context, k8s kubernetes.Interface, dyn dynamic.Interface) {
	t.Helper()
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	kubevirtNS, kvObj, _, err := findKubeVirtWithVSOCKFeatureGate(checkCtx, dyn)
	if err != nil {
		t.Fatalf("VSOCK preflight: no KubeVirt CR with VSOCK feature gate found.\n"+
			"Tried namespaces %s.\n"+
			"%v\n"+
			"Remediation: install/configure KubeVirt/CNV and add \"VSOCK\" to spec.configuration.developerConfiguration.featureGates on a KubeVirt CR (see KubeVirt docs for vsock).",
			strings.Join(kubeVirtInstallNamespaces(), ", "), err)
	}

	if ok, diag := virtHandlerHostVsockVolumesLookUsable(checkCtx, t, k8s); !ok {
		t.Fatalf("VSOCK preflight: virt-handler does not show usable vsock plumbing evidence.\n"+
			"KubeVirt CR %q/%q has VSOCK feature gate enabled, but virt-handler pod evidence checks failed.\n"+
			"%s\n"+
			"Remediation: ensure worker nodes provide /dev/vhost-vsock (or equivalent) and virt-handler pods expose either "+
			"an explicit vsock hostPath or the CNV libvirt runtime hostPath (/var/run/kubevirt-libvirt-runtimes); "+
			"guest roxagent requires a working virtio-vsock channel (/dev/vsock in the guest once the VMI is created).",
			kubevirtNS, kvObj.GetName(), diag)
	}
}

func kubeVirtInstallNamespaces() []string {
	return []string{"openshift-cnv", "kubevirt", "openshift-kubevirt"}
}

// findKubeVirtWithVSOCKFeatureGate scans every KubeVirt CR in candidate install namespaces and returns
// the first whose developerConfiguration.featureGates lists VSOCK (case-insensitive). If none qualify,
// the error message aggregates per-CR diagnostics for CI triage.
func findKubeVirtWithVSOCKFeatureGate(ctx context.Context, dyn dynamic.Interface) (ns string, obj unstructured.Unstructured, fg []string, err error) {
	var diag strings.Builder
	var lastListErr error
	for _, candidate := range kubeVirtInstallNamespaces() {
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
		return "", unstructured.Unstructured{}, nil, fmt.Errorf("last list error: %v\n%s", lastListErr, summary)
	}
	return "", unstructured.Unstructured{}, nil, fmt.Errorf("%s", summary)
}

func virtHandlerHostVsockVolumesLookUsable(ctx context.Context, t *testing.T, k8s kubernetes.Interface) (bool, string) {
	t.Helper()
	var diag strings.Builder
	selectors := []string{
		"kubevirt.io=virt-handler",
		"app.kubernetes.io/component=virt-handler",
	}
	for _, ns := range kubeVirtInstallNamespaces() {
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
