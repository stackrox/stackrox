//go:build poc_vsock_pull

package poc

import (
	"os"
	"testing"

	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"github.com/stackrox/rox/sensor/kubernetes/virtualmachine/vsockdialer"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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

	dialer := vsockdialer.NewMultiDialer(config)
	stream, err := dialer.Dial(ns, name, vsockclient.DefaultVSOCKPort, false)
	require.NoError(t, err, "VSOCK dial failed — check VMI is running and has autoattachVSOCK: true")

	report, err := vsockclient.ReadVMReport(stream)
	require.NoError(t, err, "failed to read VMReport from stream")

	t.Logf("SUCCESS: received VMReport via pull mode")
	t.Logf("  vsock_cid: %s", report.GetIndexReport().GetVsockCid())
	t.Logf("  packages:  %d", len(report.GetIndexReport().GetIndexV4().GetContents().GetPackages()))
	t.Logf("  success:   %v", report.GetIndexReport().GetIndexV4().GetSuccess())
}
