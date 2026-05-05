//go:build test_e2e

package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestVirtHandlerHostVsockVolumesLookUsable_AcceptsExplicitVsockHostPath(t *testing.T) {
	t.Parallel()
	client := kubefake.NewSimpleClientset(
		makeVirtHandlerPod("openshift-cnv", "virt-handler-1", coreV1.PodRunning, "/dev/vhost-vsock"),
	)
	ok, diag := virtHandlerHostVsockVolumesLookUsable(context.Background(), t, client)
	require.Truef(t, ok, "expected explicit vsock hostPath to pass preflight; diagnostics: %s", diag)
}

func TestVirtHandlerHostVsockVolumesLookUsable_AcceptsCNVLibvirtRuntimesHostPath(t *testing.T) {
	t.Parallel()
	client := kubefake.NewSimpleClientset(
		makeVirtHandlerPod("openshift-cnv", "virt-handler-1", coreV1.PodRunning, "/var/run/kubevirt-libvirt-runtimes"),
	)
	ok, diag := virtHandlerHostVsockVolumesLookUsable(context.Background(), t, client)
	require.Truef(t, ok, "expected CNV libvirt-runtimes hostPath to pass preflight; diagnostics: %s", diag)
}
