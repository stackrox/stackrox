//go:build test

package vmhelpers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestRenderCloudInit_InjectsAuthorizedKey(t *testing.T) {
	data, err := RenderCloudInit(VMRequest{
		GuestUser:    "cloud-user",
		SSHPublicKey: "ssh-rsa AAAATEST",
	})
	require.NoError(t, err)
	require.Contains(t, string(data), "ssh_authorized_keys:")
	require.Contains(t, string(data), "ssh-rsa AAAATEST")
}

func TestRenderCloudInit_IncludesGuestUser(t *testing.T) {
	data, err := RenderCloudInit(VMRequest{
		GuestUser:    "cloud-user",
		SSHPublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA",
	})
	require.NoError(t, err)
	require.Contains(t, string(data), "cloud-user")
	require.Contains(t, string(data), "name:")
}

func TestRenderCloudInit_GrantsPasswordlessSudo(t *testing.T) {
	data, err := RenderCloudInit(VMRequest{
		GuestUser:    "cloud-user",
		SSHPublicKey: "ssh-rsa AAAATEST",
	})
	require.NoError(t, err)
	require.Contains(t, string(data), "NOPASSWD:ALL")
}

func TestRenderCloudInit_MissingGuestUser(t *testing.T) {
	_, err := RenderCloudInit(VMRequest{
		GuestUser:    "",
		SSHPublicKey: "ssh-rsa AAAA",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "GuestUser")
}

func TestRenderCloudInit_MissingSSHPublicKey(t *testing.T) {
	_, err := RenderCloudInit(VMRequest{
		GuestUser:    "cloud-user",
		SSHPublicKey: "",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "SSHPublicKey")
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
