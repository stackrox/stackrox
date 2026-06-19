//go:build test

package vmhelpers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubevirtv1 "kubevirt.io/api/core/v1"
)

func TestRenderCloudInit_Content(t *testing.T) {
	t.Parallel()

	output := renderCloudInit(VMRequest{
		GuestUser:    "cloud-user",
		SSHPublicKey: "ssh-rsa AAAATEST",
	})
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

func TestCreateVirtualMachine_RequiresRequiredFields(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		req     VMRequest
		wantErr string
	}{
		"missing Name": {
			req: VMRequest{
				Namespace:    "ns",
				Image:        "quay.io/example/rhel9:latest",
				GuestUser:    "cloud-user",
				SSHPublicKey: "ssh-rsa AAAA",
			},
			wantErr: "VMRequest Name, Namespace, Image, GuestUser, and SSHPublicKey are required",
		},
		"missing Namespace": {
			req: VMRequest{
				Name:         "vm",
				Image:        "quay.io/example/rhel9:latest",
				GuestUser:    "cloud-user",
				SSHPublicKey: "ssh-rsa AAAA",
			},
			wantErr: "VMRequest Name, Namespace, Image, GuestUser, and SSHPublicKey are required",
		},
		"missing Image": {
			req: VMRequest{
				Name:         "vm",
				Namespace:    "ns",
				GuestUser:    "cloud-user",
				SSHPublicKey: "ssh-rsa AAAA",
			},
			wantErr: "VMRequest Name, Namespace, Image, GuestUser, and SSHPublicKey are required",
		},
		"missing GuestUser": {
			req: VMRequest{
				Name:         "vm",
				Namespace:    "ns",
				Image:        "quay.io/example/rhel9:latest",
				SSHPublicKey: "ssh-rsa AAAA",
			},
			wantErr: "VMRequest Name, Namespace, Image, GuestUser, and SSHPublicKey are required",
		},
		"missing SSHPublicKey": {
			req: VMRequest{
				Name:      "vm",
				Namespace: "ns",
				Image:     "quay.io/example/rhel9:latest",
				GuestUser: "cloud-user",
			},
			wantErr: "VMRequest Name, Namespace, Image, GuestUser, and SSHPublicKey are required",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := CreateVirtualMachine(context.Background(), nil, tt.req)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestCreateVirtualMachine_SetsDefaultMemoryRequest(t *testing.T) {
	t.Parallel()
	client := &captureDynamicClient{}
	req := VMRequest{
		Name:         "vm-rhel9",
		Namespace:    "ns",
		Image:        "registry.redhat.io/rhel9/rhel-guest-image",
		GuestUser:    "cloud-user",
		SSHPublicKey: "ssh-rsa AAAA",
	}

	err := CreateVirtualMachine(context.Background(), client, req)
	require.NoError(t, err)
	require.Equal(t, vmGVR, client.resource)
	require.Equal(t, req.Namespace, client.namespace)
	require.NotNil(t, client.created)

	got, found, err := unstructured.NestedString(client.created.Object, "spec", "template", "spec", "domain", "resources", "requests", "memory")
	require.NoError(t, err)
	require.True(t, found, "expected memory request in VM manifest")
	require.Equal(t, defaultVMMemoryRequest, got)

	vsockEnabled, found, err := unstructured.NestedBool(client.created.Object, "spec", "template", "spec", "domain", "devices", "autoattachVSOCK")
	require.NoError(t, err)
	require.True(t, found, "expected autoattachVSOCK in VM manifest")
	require.True(t, vsockEnabled, "expected autoattachVSOCK=true in VM manifest")

	cores, found, err := unstructured.NestedFieldNoCopy(client.created.Object, "spec", "template", "spec", "domain", "cpu", "cores")
	require.NoError(t, err)
	require.True(t, found, "expected domain.cpu.cores in VM manifest")
	require.EqualValues(t, uint64(3), cores, "expected VM cpu cores to be 3")
}

type captureDynamicClient struct {
	resource  schema.GroupVersionResource
	namespace string
	created   *unstructured.Unstructured
}

func (c *captureDynamicClient) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return &captureResourceClient{
		parent:   c,
		resource: resource,
	}
}

type captureResourceClient struct {
	parent    *captureDynamicClient
	resource  schema.GroupVersionResource
	namespace string
}

func (c *captureResourceClient) Namespace(namespace string) dynamic.ResourceInterface {
	return &captureResourceClient{
		parent:    c.parent,
		resource:  c.resource,
		namespace: namespace,
	}
}

func (c *captureResourceClient) Create(_ context.Context, obj *unstructured.Unstructured, _ metav1.CreateOptions, _ ...string) (*unstructured.Unstructured, error) {
	c.parent.resource = c.resource
	c.parent.namespace = c.namespace
	c.parent.created = obj
	return obj, nil
}

func (*captureResourceClient) Update(context.Context, *unstructured.Unstructured, metav1.UpdateOptions, ...string) (*unstructured.Unstructured, error) {
	return nil, errors.New("unexpected Update call")
}

func (*captureResourceClient) UpdateStatus(context.Context, *unstructured.Unstructured, metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	return nil, errors.New("unexpected UpdateStatus call")
}

func (*captureResourceClient) Delete(context.Context, string, metav1.DeleteOptions, ...string) error {
	return errors.New("unexpected Delete call")
}

func (*captureResourceClient) DeleteCollection(context.Context, metav1.DeleteOptions, metav1.ListOptions) error {
	return errors.New("unexpected DeleteCollection call")
}

func (*captureResourceClient) Get(context.Context, string, metav1.GetOptions, ...string) (*unstructured.Unstructured, error) {
	return nil, errors.New("unexpected Get call")
}

func (*captureResourceClient) List(context.Context, metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return nil, errors.New("unexpected List call")
}

func (*captureResourceClient) Watch(context.Context, metav1.ListOptions) (watch.Interface, error) {
	return nil, errors.New("unexpected Watch call")
}

func (*captureResourceClient) Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*unstructured.Unstructured, error) {
	return nil, errors.New("unexpected Patch call")
}

func (*captureResourceClient) Apply(context.Context, string, *unstructured.Unstructured, metav1.ApplyOptions, ...string) (*unstructured.Unstructured, error) {
	return nil, errors.New("unexpected Apply call")
}

func (*captureResourceClient) ApplyStatus(context.Context, string, *unstructured.Unstructured, metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	return nil, errors.New("unexpected ApplyStatus call")
}

func TestVMFailureConditionDetail(t *testing.T) {
	t.Parallel()

	vm := &kubevirtv1.VirtualMachine{
		Status: kubevirtv1.VirtualMachineStatus{
			Conditions: []kubevirtv1.VirtualMachineCondition{
				{
					Type:    kubevirtv1.VirtualMachineFailure,
					Status:  coreV1.ConditionTrue,
					Reason:  "FailedCreate",
					Message: "no memory requested",
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
