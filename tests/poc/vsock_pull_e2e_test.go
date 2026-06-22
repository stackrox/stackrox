//go:build poc_vsock_pull

package poc

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
)

// TestVSOCKPull validates the full pull path:
// Sensor → virt-api → virt-handler → vsock into VM → roxagent serve → VMReport response.
//
// Prerequisites:
//   - KubeVirt cluster with VSOCK feature gate enabled
//   - A running VMI with `roxagent serve --port 818`
//   - KUBECONFIG set or running in-cluster
//   - Sensor's SA (or current kubeconfig) has virtualmachineinstances/vsock permission
//   - Env vars: POC_VMI_NAMESPACE, POC_VMI_NAME
//
// Run: go test -tags poc_vsock_pull -v ./tests/poc/ -run TestVSOCKPull -timeout 60s
func TestVSOCKPull(t *testing.T) {
	ns := os.Getenv("POC_VMI_NAMESPACE")
	name := os.Getenv("POC_VMI_NAME")
	if ns == "" || name == "" {
		t.Skip("POC_VMI_NAMESPACE and POC_VMI_NAME must be set")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		t.Logf("Not in cluster, trying KUBECONFIG...")
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(), nil,
		).ClientConfig()
		require.NoError(t, err, "failed to get kubeconfig")
	}

	// ponytail: Using the typed KubeVirt client directly instead of kubecli
	// to avoid transitive dep issues (kubecli's generated mock pulls
	// k8s.io/client-go/kubernetes/typed/storagemigration/v1alpha1).
	// The generated typed client stubs VSOCK with "not implemented" — when
	// kubevirt dep alignment is fixed, switch to kubecli for real E2E runs.
	kvClient, err := kvcorev1.NewForConfig(config)
	require.NoError(t, err, "failed to create KubeVirt typed client")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = ctx // ponytail: Dial doesn't accept ctx yet; timeout is a follow-up.

	dialer := vsockclient.NewDialer(kvClient.VirtualMachineInstances(ns))
	stream, err := dialer.Dial(name, vsockclient.DefaultVSOCKPort, false)
	require.NoError(t, err, "VSOCK dial failed — check VMI is running and has autoattachVSOCK: true")

	report, err := vsockclient.ReadVMReport(stream)
	require.NoError(t, err, "failed to read VMReport from stream")

	t.Logf("SUCCESS: received VMReport via pull mode")
	t.Logf("  vsock_cid: %s", report.GetIndexReport().GetVsockCid())
	t.Logf("  packages:  %d", len(report.GetIndexReport().GetIndexV4().GetContents().GetPackages()))
	t.Logf("  success:   %v", report.GetIndexReport().GetIndexV4().GetSuccess())
}
