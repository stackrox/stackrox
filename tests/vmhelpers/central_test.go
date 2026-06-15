//go:build test

package vmhelpers

import (
	"context"
	"errors"
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRawListQueryNamespaceAndName(t *testing.T) {
	q := rawListQueryNamespaceAndName("stackrox", "vm-rhel9")
	require.Contains(t, q, `Namespace:"stackrox"`)
	require.Contains(t, q, `Virtual Machine Name:"vm-rhel9"`)

	q = rawListQueryNamespaceAndName("stack rox", "vm:special+name")
	require.Contains(t, q, `Namespace:"stack rox"`)
	require.Contains(t, q, `Virtual Machine Name:"vm:special+name"`)
}

type stubVirtualMachineClient struct {
	listFn func(ctx context.Context, req *v2.ListVirtualMachinesRequest) (*v2.ListVirtualMachinesResponse, error)
	getFn  func(ctx context.Context, req *v2.GetVirtualMachineRequest) (*v2.VirtualMachine, error)
}

func (s *stubVirtualMachineClient) ListVirtualMachines(ctx context.Context, in *v2.ListVirtualMachinesRequest, _ ...grpc.CallOption) (*v2.ListVirtualMachinesResponse, error) {
	if s.listFn == nil {
		return &v2.ListVirtualMachinesResponse{}, nil
	}
	return s.listFn(ctx, in)
}

func (s *stubVirtualMachineClient) GetVirtualMachine(ctx context.Context, in *v2.GetVirtualMachineRequest, _ ...grpc.CallOption) (*v2.VirtualMachine, error) {
	if s.getFn == nil {
		return nil, errors.New("get not stubbed")
	}
	return s.getFn(ctx, in)
}

func TestWaitForVMPresentInCentral_UsesListVirtualMachines(t *testing.T) {
	ctx := context.Background()
	opts := WaitOptions{Timeout: 200 * time.Millisecond, PollInterval: 5 * time.Millisecond}
	var sawQuery string
	client := &stubVirtualMachineClient{
		listFn: func(ctx context.Context, req *v2.ListVirtualMachinesRequest) (*v2.ListVirtualMachinesResponse, error) {
			sawQuery = req.GetQuery().GetQuery()
			return &v2.ListVirtualMachinesResponse{
				VirtualMachines: []*v2.VirtualMachine{
					{Id: "id-1", Namespace: "ns1", Name: "vm1"},
				},
			}, nil
		},
	}
	vm, err := WaitForVMPresentInCentral(ctx, client, opts, "ns1", "vm1")
	require.NoError(t, err)
	require.Equal(t, "id-1", vm.GetId())
	require.NotEmpty(t, sawQuery)
	require.Contains(t, sawQuery, `Namespace:"ns1"`)
}

func TestWaitForVMScanTimestamp(t *testing.T) {
	ctx := context.Background()
	opts := WaitOptions{Timeout: 150 * time.Millisecond, PollInterval: 5 * time.Millisecond}
	var calls int
	client := &stubVirtualMachineClient{
		getFn: func(ctx context.Context, req *v2.GetVirtualMachineRequest) (*v2.VirtualMachine, error) {
			calls++
			if calls < 3 {
				return &v2.VirtualMachine{
					Id:   req.GetId(),
					Scan: &v2.VirtualMachineScan{},
				}, nil
			}
			return &v2.VirtualMachine{
				Id: req.GetId(),
				Scan: &v2.VirtualMachineScan{
					ScanTime: timestamppb.Now(),
				},
			}, nil
		},
	}
	vm, err := WaitForVMScanTimestamp(ctx, client, opts, "vid")
	require.NoError(t, err)
	require.NotNil(t, vm.GetScan().GetScanTime())
	require.GreaterOrEqual(t, calls, 3)
}

func TestCentralWaitStubClients(t *testing.T) {
	ctx := context.Background()
	opts := WaitOptions{Timeout: 150 * time.Millisecond, PollInterval: 5 * time.Millisecond}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "WaitForVMIdentityFields",
			run: func(t *testing.T) {
				var calls int
				client := &stubVirtualMachineClient{
					getFn: func(_ context.Context, _ *v2.GetVirtualMachineRequest) (*v2.VirtualMachine, error) {
						calls++
						if calls == 1 {
							return &v2.VirtualMachine{Id: "vid", Namespace: "a", Name: "b"}, nil
						}
						return &v2.VirtualMachine{Id: "vid", Namespace: "ns-x", Name: "vm-x"}, nil
					},
				}
				vm, err := WaitForVMIdentityFields(ctx, client, opts, "vid", "ns-x", "vm-x")
				require.NoError(t, err)
				require.Equal(t, "ns-x", vm.GetNamespace())
				require.Equal(t, "vm-x", vm.GetName())
				require.GreaterOrEqual(t, calls, 2)
			},
		},
		{
			name: "WaitForVMRunningInCentral",
			run: func(t *testing.T) {
				var calls int
				client := &stubVirtualMachineClient{
					getFn: func(_ context.Context, _ *v2.GetVirtualMachineRequest) (*v2.VirtualMachine, error) {
						calls++
						if calls < 2 {
							return &v2.VirtualMachine{Id: "r1", State: v2.VirtualMachine_STOPPED}, nil
						}
						return &v2.VirtualMachine{Id: "r1", State: v2.VirtualMachine_RUNNING}, nil
					},
				}
				vm, err := WaitForVMRunningInCentral(ctx, client, opts, "r1")
				require.NoError(t, err)
				require.Equal(t, v2.VirtualMachine_RUNNING, vm.GetState())
				require.GreaterOrEqual(t, calls, 2)
			},
		},
		{
			name: "WaitForVMScanNonNil",
			run: func(t *testing.T) {
				var calls int
				client := &stubVirtualMachineClient{
					getFn: func(_ context.Context, req *v2.GetVirtualMachineRequest) (*v2.VirtualMachine, error) {
						calls++
						if calls == 1 {
							return &v2.VirtualMachine{Id: req.GetId(), Scan: nil}, nil
						}
						return &v2.VirtualMachine{Id: req.GetId(), Scan: &v2.VirtualMachineScan{}}, nil
					},
				}
				vm, err := WaitForVMScanNonNil(ctx, client, opts, "s1")
				require.NoError(t, err)
				require.NotNil(t, vm.GetScan())
				require.GreaterOrEqual(t, calls, 2)
			},
		},
	} {
		t.Run(tc.name, tc.run)
	}
}
