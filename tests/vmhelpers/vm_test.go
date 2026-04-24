//go:build test

package vmhelpers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestRenderCloudInit_Content(t *testing.T) {
	t.Parallel()
	data, err := RenderCloudInit(VMRequest{
		GuestUser:    "cloud-user",
		SSHPublicKey: "ssh-rsa AAAATEST",
	})
	require.NoError(t, err)
	output := string(data)
	for _, want := range []string{
		"ssh_authorized_keys:",
		"ssh-rsa AAAATEST",
		"cloud-user",
		"name:",
		"NOPASSWD:ALL",
	} {
		require.Contains(t, output, want)
	}
}

func TestRenderCloudInit_MissingFields(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		req     VMRequest
		wantErr string
	}{
		"missing GuestUser": {
			req:     VMRequest{SSHPublicKey: "ssh-rsa AAAA"},
			wantErr: "GuestUser",
		},
		"missing SSHPublicKey": {
			req:     VMRequest{GuestUser: "cloud-user"},
			wantErr: "SSHPublicKey",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := RenderCloudInit(tt.req)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestCreateVirtualMachine_RequiresCloudInitFields(t *testing.T) {
	err := CreateVirtualMachine(context.Background(), nil, VMRequest{
		Name: "vm", Namespace: "ns", Image: "quay.io/example/rhel9:latest",
		GuestUser: "", SSHPublicKey: "ssh-rsa AAAA",
	})
	require.ErrorContains(t, err, "GuestUser")

	err = CreateVirtualMachine(context.Background(), nil, VMRequest{
		Name: "vm", Namespace: "ns", Image: "quay.io/example/rhel9:latest",
		GuestUser: "cloud-user", SSHPublicKey: "",
	})
	require.ErrorContains(t, err, "SSHPublicKey")
}

func TestCreateVirtualMachine_SetsDefaultMemoryRequest(t *testing.T) {
	t.Parallel()
	client := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	req := VMRequest{
		Name:         "vm-rhel9",
		Namespace:    "ns",
		Image:        "registry.redhat.io/rhel9/rhel-guest-image",
		GuestUser:    "cloud-user",
		SSHPublicKey: "ssh-rsa AAAA",
	}

	err := CreateVirtualMachine(context.Background(), client, req)
	require.NoError(t, err)

	obj, err := client.Resource(vmGVR).Namespace(req.Namespace).Get(context.Background(), req.Name, metav1.GetOptions{})
	require.NoError(t, err)
	got, found, err := unstructured.NestedString(obj.Object, "spec", "template", "spec", "domain", "resources", "requests", "memory")
	require.NoError(t, err)
	require.True(t, found, "expected memory request in VM manifest")
	require.Equal(t, defaultVMMemoryRequest, got)

	vsockEnabled, found, err := unstructured.NestedBool(obj.Object, "spec", "template", "spec", "domain", "devices", "autoattachVSOCK")
	require.NoError(t, err)
	require.True(t, found, "expected autoattachVSOCK in VM manifest")
	require.True(t, vsockEnabled, "expected autoattachVSOCK=true in VM manifest")

	cores, found, err := unstructured.NestedInt64(obj.Object, "spec", "template", "spec", "domain", "cpu", "cores")
	require.NoError(t, err)
	require.True(t, found, "expected domain.cpu.cores in VM manifest")
	require.EqualValues(t, 3, cores, "expected VM cpu cores to be 3")
}

func TestVMFailureConditionDetail(t *testing.T) {
	t.Parallel()
	vm := &unstructured.Unstructured{
		Object: map[string]any{
			"status": map[string]any{
				"conditions": []any{
					map[string]any{
						"type":    "Failure",
						"status":  "True",
						"reason":  "FailedCreate",
						"message": "no memory requested",
					},
				},
			},
		},
	}
	detail, terminal := vmFailureConditionDetail(vm)
	require.True(t, terminal)
	require.Contains(t, detail, "FailedCreate")
	require.Contains(t, detail, "no memory requested")
}

func TestWaitForVirtualMachineInstanceRunning_FailsFastOnTerminalVMIPhase(t *testing.T) {
	t.Parallel()

	vmi := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachineInstance",
			"metadata": map[string]any{
				"namespace": "ns",
				"name":      "vm-rhel9",
			},
			"status": map[string]any{
				"phase": "Failed",
			},
		},
	}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			vmiGVR: "VirtualMachineInstanceList",
		},
		vmi,
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := WaitForVirtualMachineInstanceRunning(t, ctx, client, "ns", "vm-rhel9")
	require.Error(t, err)
	require.ErrorContains(t, err, "VMI reached terminal phase")
}
