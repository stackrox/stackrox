//go:build test_e2e

package tests

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func makeVirtHandlerPod(namespace, name string, phase coreV1.PodPhase, hostPaths ...string) *coreV1.Pod {
	volumes := make([]coreV1.Volume, 0, len(hostPaths))
	for i, p := range hostPaths {
		volumes = append(volumes, coreV1.Volume{
			Name: fmt.Sprintf("hostpath-%d", i),
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{Path: p},
			},
		})
	}
	return &coreV1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				"kubevirt.io": "virt-handler",
			},
		},
		Spec: coreV1.PodSpec{
			Volumes: volumes,
		},
		Status: coreV1.PodStatus{
			Phase: phase,
		},
	}
}

func makeNode(name string, labels map[string]string, unschedulable bool, kvmCapacity string) *coreV1.Node {
	capacity := coreV1.ResourceList{}
	if kvmCapacity != "" {
		capacity[coreV1.ResourceName("devices.kubevirt.io/kvm")] = resource.MustParse(kvmCapacity)
	}

	return &coreV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: coreV1.NodeSpec{
			Unschedulable: unschedulable,
		},
		Status: coreV1.NodeStatus{
			Capacity: capacity,
		},
	}
}

func TestInspectClusterKVMReadiness(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		nodes        []*coreV1.Node
		wantUsable   bool
		wantScope    string
		wantFallback bool
		wantChecked  []kvmPreflightNode
	}{
		"should report success when a schedulable worker advertises KVM capacity": {
			nodes: []*coreV1.Node{
				makeNode("worker-a", map[string]string{"node-role.kubernetes.io/worker": ""}, false, "1"),
			},
			wantUsable: true,
			wantScope:  workerNodeScope,
			wantChecked: []kvmPreflightNode{
				{Name: "worker-a", Unschedulable: false, KVMCapacity: "1", Eligible: true},
			},
		},
		"should fail when all worker nodes advertise zero KVM capacity": {
			nodes: []*coreV1.Node{
				makeNode("worker-a", map[string]string{"node-role.kubernetes.io/worker": ""}, false, "0"),
				makeNode("worker-b", map[string]string{"node-role.kubernetes.io/worker": ""}, false, "0"),
			},
			wantUsable: false,
			wantScope:  workerNodeScope,
			wantChecked: []kvmPreflightNode{
				{Name: "worker-a", Unschedulable: false, KVMCapacity: "0", Eligible: false},
				{Name: "worker-b", Unschedulable: false, KVMCapacity: "0", Eligible: false},
			},
		},
		"should fail when only unschedulable workers advertise KVM capacity": {
			nodes: []*coreV1.Node{
				makeNode("worker-a", map[string]string{"node-role.kubernetes.io/worker": ""}, true, "1"),
			},
			wantUsable: false,
			wantScope:  workerNodeScope,
			wantChecked: []kvmPreflightNode{
				{Name: "worker-a", Unschedulable: true, KVMCapacity: "1", Eligible: false},
			},
		},
		"should fall back to all nodes when worker labels are absent": {
			nodes: []*coreV1.Node{
				makeNode("node-a", nil, false, "1"),
			},
			wantUsable:   true,
			wantScope:    kvmAllSchedulableNodeScope,
			wantFallback: true,
			wantChecked: []kvmPreflightNode{
				{Name: "node-a", Unschedulable: false, KVMCapacity: "1", Eligible: true},
			},
		},
		"should skip unschedulable nodes in fallback scope": {
			nodes: []*coreV1.Node{
				makeNode("node-a", nil, true, "1"),
				makeNode("node-b", nil, false, "1"),
			},
			wantUsable:   true,
			wantScope:    kvmAllSchedulableNodeScope,
			wantFallback: true,
			wantChecked: []kvmPreflightNode{
				{Name: "node-b", Unschedulable: false, KVMCapacity: "1", Eligible: true},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			objects := make([]runtime.Object, 0, len(tc.nodes))
			for _, node := range tc.nodes {
				objects = append(objects, node)
			}

			client := kubefake.NewSimpleClientset(objects...)
			result, err := inspectClusterKVMReadiness(t.Context(), client)
			require.NoError(t, err)
			require.Equal(t, tc.wantUsable, result.IsUsable())
			require.Equal(t, tc.wantScope, result.Scope)
			require.Equal(t, tc.wantFallback, result.UsedAllSchedulableNodesFallback)
			require.Equal(t, tc.wantChecked, result.CheckedNodes)
		})
	}
}

func TestClusterKVMPreflightResultDiagnostic(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		result   clusterKVMPreflightResult
		wantText []string
	}{
		"should format failure diagnostics with checked nodes": {
			result: clusterKVMPreflightResult{
				Scope: workerNodeScope,
				CheckedNodes: []kvmPreflightNode{
					{Name: "worker-a", Unschedulable: false, KVMCapacity: "0", Eligible: false},
					{Name: "worker-b", Unschedulable: true, KVMCapacity: "1", Eligible: false},
				},
			},
			wantText: []string{
				"No worker-labeled nodes with devices.kubevirt.io/kvm > 0.",
				"worker-a(unschedulable=false kvm=0)",
				"worker-b(unschedulable=true kvm=1)",
			},
		},
		"should format fallback and success diagnostics separately": {
			result: clusterKVMPreflightResult{
				Scope:                           kvmAllSchedulableNodeScope,
				UsedAllSchedulableNodesFallback: true,
				CheckedNodes: []kvmPreflightNode{
					{Name: "node-a", Unschedulable: false, KVMCapacity: "1", Eligible: true},
				},
			},
			wantText: []string{
				kvmFallbackDiagnostic,
				"KVM-capable all schedulable nodes: node-a(kvm=1)",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			diag := tc.result.Diagnostic()
			for _, want := range tc.wantText {
				require.Contains(t, diag, want)
			}
		})
	}
}

func TestVirtHandlerHostVsockVolumesLookUsable(t *testing.T) {
	t.Parallel()
	for name, hostPath := range map[string]string{
		"explicit vsock hostPath":       "/dev/vhost-vsock",
		"CNV libvirt-runtimes hostPath": "/var/run/kubevirt-libvirt-runtimes",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client := kubefake.NewSimpleClientset(
				makeVirtHandlerPod("openshift-cnv", "virt-handler-1", coreV1.PodRunning, hostPath),
			)
			ok, diag := virtHandlerHostVsockVolumesLookUsable(context.Background(), t, client)
			require.Truef(t, ok, "expected %s to pass preflight; diagnostics: %s", hostPath, diag)
		})
	}
}

func TestVerifyClusterVSOCKReadyPhases_ShouldGivePhaseTwoAFreshTimeoutAfterSlowPhaseOne(t *testing.T) {
	t.Parallel()

	const (
		featureGateTimeout = 150 * time.Millisecond
		virtHandlerTimeout = 150 * time.Millisecond
		phaseOneDelay      = 100 * time.Millisecond
	)

	var phaseTwoRemaining time.Duration
	ref, diag, err := verifyClusterVSOCKReadyPhases(
		t.Context(),
		featureGateTimeout,
		virtHandlerTimeout,
		func(context.Context) (kubeVirtVSOCKRef, error) {
			time.Sleep(phaseOneDelay)
			return kubeVirtVSOCKRef{
				Namespace: "openshift-cnv",
				Name:      "kubevirt",
			}, nil
		},
		func(ctx context.Context) (string, error) {
			deadline, ok := ctx.Deadline()
			require.True(t, ok, "phase two should have its own timeout context")
			phaseTwoRemaining = time.Until(deadline)
			return "virt-handler ready", nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, kubeVirtVSOCKRef{
		Namespace: "openshift-cnv",
		Name:      "kubevirt",
	}, ref)
	require.Equal(t, "virt-handler ready", diag)
	require.Greater(t, phaseTwoRemaining, 75*time.Millisecond,
		"phase two should receive a fresh timeout budget instead of the leftovers from phase one")
}

func TestVerifyClusterVSOCKReadyPhases_ShouldReturnRefOnPhaseTwoFailure(t *testing.T) {
	t.Parallel()

	wantRef := kubeVirtVSOCKRef{Namespace: "openshift-cnv", Name: "kubevirt"}
	wantDiag := "virt-handler volumes look wrong"
	wantErr := errors.New("phase two verification failed")

	ref, diag, err := verifyClusterVSOCKReadyPhases(
		t.Context(),
		time.Second,
		time.Second,
		func(context.Context) (kubeVirtVSOCKRef, error) {
			return wantRef, nil
		},
		func(context.Context) (string, error) {
			return wantDiag, wantErr
		},
	)
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, wantRef, ref, "ref from phase one must be preserved on phase two failure")
	require.Equal(t, wantDiag, diag, "diagnostic from phase two must be returned on failure")
}

func TestVerifyClusterVSOCKReadyPhases_ShouldStopBeforePhaseTwoWhenPhaseOneFails(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("phase one failed")
	phaseTwoCalled := false

	_, _, err := verifyClusterVSOCKReadyPhases(
		t.Context(),
		time.Second,
		time.Second,
		func(context.Context) (kubeVirtVSOCKRef, error) {
			return kubeVirtVSOCKRef{}, wantErr
		},
		func(context.Context) (string, error) {
			phaseTwoCalled = true
			return "", nil
		},
	)
	require.ErrorIs(t, err, wantErr)
	require.False(t, phaseTwoCalled, "phase two should not run after a phase one failure")
}
