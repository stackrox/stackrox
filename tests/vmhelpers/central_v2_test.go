//go:build test && !test_e2e && !test_e2e_vm

package vmhelpers

import (
	"context"
	"errors"
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type stubV2Client struct {
	listVMsFn            func(ctx context.Context, req *v2.ListVMsRequest) (*v2.ListVMsResponse, error)
	getVMFn              func(ctx context.Context, req *v2.GetVMRequest) (*v2.VMDetail, error)
	listVMComponentsFn   func(ctx context.Context, req *v2.ListVMComponentsRequest) (*v2.ListVMComponentsResponse, error)
	listVMCVEsByVMFn     func(ctx context.Context, req *v2.ListVMCVEsByVMRequest) (*v2.ListVMCVEsByVMResponse, error)
	getVMCVEComponentsFn func(ctx context.Context, req *v2.GetVMCVEComponentsRequest) (*v2.GetVMCVEComponentsResponse, error)
	getVMVulnSummaryFn   func(ctx context.Context, req *v2.GetVMVulnSummaryRequest) (*v2.VMVulnSummary, error)
}

func (s *stubV2Client) GetVM(ctx context.Context, in *v2.GetVMRequest, _ ...grpc.CallOption) (*v2.VMDetail, error) {
	if s.getVMFn == nil {
		return nil, errors.New("GetVM not stubbed")
	}
	return s.getVMFn(ctx, in)
}

func (s *stubV2Client) GetVMVulnSummary(ctx context.Context, in *v2.GetVMVulnSummaryRequest, _ ...grpc.CallOption) (*v2.VMVulnSummary, error) {
	if s.getVMVulnSummaryFn == nil {
		return &v2.VMVulnSummary{}, nil
	}
	return s.getVMVulnSummaryFn(ctx, in)
}

func (s *stubV2Client) ListVMCVEsByVM(ctx context.Context, in *v2.ListVMCVEsByVMRequest, _ ...grpc.CallOption) (*v2.ListVMCVEsByVMResponse, error) {
	if s.listVMCVEsByVMFn == nil {
		return &v2.ListVMCVEsByVMResponse{}, nil
	}
	return s.listVMCVEsByVMFn(ctx, in)
}

func (s *stubV2Client) GetVMCVEComponents(ctx context.Context, in *v2.GetVMCVEComponentsRequest, _ ...grpc.CallOption) (*v2.GetVMCVEComponentsResponse, error) {
	if s.getVMCVEComponentsFn == nil {
		return &v2.GetVMCVEComponentsResponse{}, nil
	}
	return s.getVMCVEComponentsFn(ctx, in)
}

func (s *stubV2Client) ListVMComponents(ctx context.Context, in *v2.ListVMComponentsRequest, _ ...grpc.CallOption) (*v2.ListVMComponentsResponse, error) {
	if s.listVMComponentsFn == nil {
		return &v2.ListVMComponentsResponse{}, nil
	}
	return s.listVMComponentsFn(ctx, in)
}

func (s *stubV2Client) ListVMs(ctx context.Context, in *v2.ListVMsRequest, _ ...grpc.CallOption) (*v2.ListVMsResponse, error) {
	if s.listVMsFn == nil {
		return &v2.ListVMsResponse{}, nil
	}
	return s.listVMsFn(ctx, in)
}

func (s *stubV2Client) ListVMCVEs(context.Context, *v2.ListVMCVEsRequest, ...grpc.CallOption) (*v2.ListVMCVEsResponse, error) {
	return &v2.ListVMCVEsResponse{}, nil
}

func (s *stubV2Client) GetVMDashboardCounts(context.Context, *v2.VMDashboardCountsRequest, ...grpc.CallOption) (*v2.VMDashboardCountsResponse, error) {
	return &v2.VMDashboardCountsResponse{}, nil
}

func (s *stubV2Client) GetVMCVEDetail(context.Context, *v2.GetVMCVEDetailRequest, ...grpc.CallOption) (*v2.VMCVEDetail, error) {
	return &v2.VMCVEDetail{}, nil
}

func (s *stubV2Client) ListVMCVEAffectedVMs(context.Context, *v2.ListVMCVEAffectedVMsRequest, ...grpc.CallOption) (*v2.ListVMCVEAffectedVMsResponse, error) {
	return &v2.ListVMCVEAffectedVMsResponse{}, nil
}

func TestWaitForV2VMPresentInCentral_UsesListVMs(t *testing.T) {
	ctx := t.Context()
	opts := WaitOptions{Timeout: 200 * time.Millisecond, PollInterval: 5 * time.Millisecond}
	var sawQuery string
	client := &stubV2Client{
		listVMsFn: func(_ context.Context, req *v2.ListVMsRequest) (*v2.ListVMsResponse, error) {
			sawQuery = req.GetQuery().GetQuery()
			return &v2.ListVMsResponse{
				Vms: []*v2.VMListItem{
					{Id: "id-1", Namespace: "ns1", Name: "vm1"},
				},
			}, nil
		},
	}
	vm, err := WaitForV2VMPresentInCentral(ctx, client, opts, "ns1", "vm1")
	require.NoError(t, err)
	require.Equal(t, "id-1", vm.GetId())
	require.Contains(t, sawQuery, `Namespace:"ns1"`)
	require.Contains(t, sawQuery, `Virtual Machine Name:"vm1"`)
}

func TestWaitForV2VMLatestScan(t *testing.T) {
	ctx := t.Context()
	opts := WaitOptions{Timeout: 150 * time.Millisecond, PollInterval: 5 * time.Millisecond}
	var calls int
	client := &stubV2Client{
		getVMFn: func(_ context.Context, req *v2.GetVMRequest) (*v2.VMDetail, error) {
			calls++
			if calls < 3 {
				return &v2.VMDetail{Id: req.GetId()}, nil
			}
			return &v2.VMDetail{
				Id: req.GetId(),
				LatestScan: &v2.VMScanInfo{
					ScanTime: timestamppb.Now(),
				},
			}, nil
		},
	}
	vm, err := WaitForV2VMLatestScan(ctx, client, opts, "vid")
	require.NoError(t, err)
	require.NotNil(t, vm.GetLatestScan().GetScanTime())
	require.GreaterOrEqual(t, calls, 3)
}

func TestWaitForV2VMRunningInCentral(t *testing.T) {
	ctx := t.Context()
	opts := WaitOptions{Timeout: 150 * time.Millisecond, PollInterval: 5 * time.Millisecond}
	var calls int
	client := &stubV2Client{
		getVMFn: func(_ context.Context, _ *v2.GetVMRequest) (*v2.VMDetail, error) {
			calls++
			if calls < 2 {
				return &v2.VMDetail{Id: "r1", State: v2.VirtualMachineV2State_VM_STATE_STOPPED}, nil
			}
			return &v2.VMDetail{Id: "r1", State: v2.VirtualMachineV2State_VM_STATE_RUNNING}, nil
		},
	}
	vm, err := WaitForV2VMRunningInCentral(ctx, client, opts, "r1")
	require.NoError(t, err)
	require.Equal(t, v2.VirtualMachineV2State_VM_STATE_RUNNING, vm.GetState())
}

func TestListAllVMCVEsByVM_Pages(t *testing.T) {
	ctx := t.Context()
	client := &stubV2Client{
		listVMCVEsByVMFn: func(_ context.Context, req *v2.ListVMCVEsByVMRequest) (*v2.ListVMCVEsByVMResponse, error) {
			switch req.GetQuery().GetPagination().GetOffset() {
			case 0:
				return &v2.ListVMCVEsByVMResponse{
					TotalCount: 2,
					Cves:       []*v2.VMCVERow{{Cve: "CVE-1"}, {Cve: "CVE-2"}},
				}, nil
			default:
				return &v2.ListVMCVEsByVMResponse{TotalCount: 2}, nil
			}
		},
	}
	cves, total, err := ListAllVMCVEsByVM(ctx, client, "vid")
	require.NoError(t, err)
	require.Equal(t, int32(2), total)
	require.Equal(t, []string{"CVE-1", "CVE-2"}, []string{cves[0].GetCve(), cves[1].GetCve()})
}

func TestListAllVMComponents_Pages(t *testing.T) {
	ctx := t.Context()
	page0 := make([]*v2.VMComponentRow, v2ListPageSize)
	for i := range page0 {
		page0[i] = &v2.VMComponentRow{Name: "pkg"}
	}
	client := &stubV2Client{
		listVMComponentsFn: func(_ context.Context, req *v2.ListVMComponentsRequest) (*v2.ListVMComponentsResponse, error) {
			switch req.GetQuery().GetPagination().GetOffset() {
			case 0:
				return &v2.ListVMComponentsResponse{TotalCount: int32(v2ListPageSize + 1), Components: page0}, nil
			case v2ListPageSize:
				return &v2.ListVMComponentsResponse{
					TotalCount: int32(v2ListPageSize + 1),
					Components: []*v2.VMComponentRow{{Name: "last"}},
				}, nil
			default:
				return &v2.ListVMComponentsResponse{TotalCount: int32(v2ListPageSize + 1)}, nil
			}
		},
	}
	comps, total, err := ListAllVMComponents(ctx, client, "vid")
	require.NoError(t, err)
	require.Equal(t, int32(v2ListPageSize+1), total)
	require.Len(t, comps, v2ListPageSize+1)
	require.Equal(t, "last", comps[len(comps)-1].GetName())
}

func TestVulnCountBySeverityTotal(t *testing.T) {
	tests := map[string]struct {
		in   *v2.VulnCountBySeverity
		want int32
	}{
		"nil":   {in: nil, want: 0},
		"empty": {in: &v2.VulnCountBySeverity{}, want: 0},
		"sums": {in: &v2.VulnCountBySeverity{
			Critical:  &v2.VulnFixableCount{Total: 3},
			Important: &v2.VulnFixableCount{Total: 2},
			Moderate:  &v2.VulnFixableCount{Total: 1},
			Low:       &v2.VulnFixableCount{Total: 4},
			Unknown:   &v2.VulnFixableCount{Total: 0},
		}, want: 10},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, VulnCountBySeverityTotal(tc.in))
		})
	}
}

func TestWaitForV2ScanMissingComponent(t *testing.T) {
	ctx := t.Context()
	opts := WaitOptions{Timeout: 400 * time.Millisecond, PollInterval: 5 * time.Millisecond}
	baseline := time.Now().Add(-time.Minute)
	var calls int
	client := &stubV2Client{
		getVMFn: func(_ context.Context, req *v2.GetVMRequest) (*v2.VMDetail, error) {
			calls++
			scanTime := baseline.Add(time.Duration(calls) * time.Second)
			return &v2.VMDetail{
				Id: req.GetId(),
				LatestScan: &v2.VMScanInfo{
					ScanTime: timestamppb.New(scanTime),
				},
			}, nil
		},
		listVMComponentsFn: func(_ context.Context, req *v2.ListVMComponentsRequest) (*v2.ListVMComponentsResponse, error) {
			if req.GetQuery().GetQuery() != "" {
				if calls < 3 {
					return &v2.ListVMComponentsResponse{
						TotalCount: 1,
						Components: []*v2.VMComponentRow{{Name: "bc"}},
					}, nil
				}
				return &v2.ListVMComponentsResponse{}, nil
			}
			return &v2.ListVMComponentsResponse{
				TotalCount: 2,
				Components: []*v2.VMComponentRow{{Name: "bash"}, {Name: "coreutils"}},
			}, nil
		},
	}
	comps, err := WaitForV2ScanMissingComponent(ctx, client, opts, "vid", "bc", baseline, 2)
	require.NoError(t, err)
	require.Len(t, comps, 2)
}

func TestWaitForV2ScanReady(t *testing.T) {
	ctx := t.Context()
	opts := WaitOptions{Timeout: 200 * time.Millisecond, PollInterval: 5 * time.Millisecond}
	var calls int
	client := &stubV2Client{
		getVMFn: func(_ context.Context, req *v2.GetVMRequest) (*v2.VMDetail, error) {
			return &v2.VMDetail{
				Id:      req.GetId(),
				GuestOs: "rhel",
				LatestScan: &v2.VMScanInfo{
					ScanTime: timestamppb.Now(),
					ScanOs:   "rhel",
				},
			}, nil
		},
		listVMComponentsFn: func(context.Context, *v2.ListVMComponentsRequest) (*v2.ListVMComponentsResponse, error) {
			calls++
			if calls == 1 {
				return &v2.ListVMComponentsResponse{
					TotalCount: 1,
					Components: []*v2.VMComponentRow{{Name: "bash", ScanStatus: v2.ScanStatus_NOT_SCANNED}},
				}, nil
			}
			return &v2.ListVMComponentsResponse{
				TotalCount: 1,
				Components: []*v2.VMComponentRow{{Name: "bash", ScanStatus: v2.ScanStatus_SCANNED}},
			}, nil
		},
	}
	vm, err := WaitForV2ScanReady(ctx, client, opts, "vid")
	require.NoError(t, err)
	require.Equal(t, "vid", vm.GetId())
	require.GreaterOrEqual(t, calls, 2)
}

func TestWaitForV2ScanReady_RetriesTransientListError(t *testing.T) {
	ctx := t.Context()
	opts := WaitOptions{Timeout: 400 * time.Millisecond, PollInterval: 5 * time.Millisecond}
	var calls int
	client := &stubV2Client{
		getVMFn: func(_ context.Context, req *v2.GetVMRequest) (*v2.VMDetail, error) {
			return &v2.VMDetail{
				Id:      req.GetId(),
				GuestOs: "rhel",
				LatestScan: &v2.VMScanInfo{
					ScanTime: timestamppb.Now(),
					ScanOs:   "rhel",
				},
			}, nil
		},
		listVMComponentsFn: func(context.Context, *v2.ListVMComponentsRequest) (*v2.ListVMComponentsResponse, error) {
			calls++
			if calls == 1 {
				return nil, status.Error(codes.Unavailable, "central busy")
			}
			return &v2.ListVMComponentsResponse{
				TotalCount: 1,
				Components: []*v2.VMComponentRow{{Name: "bash", ScanStatus: v2.ScanStatus_SCANNED}},
			}, nil
		},
	}
	vm, err := WaitForV2ScanReady(ctx, client, opts, "vid")
	require.NoError(t, err)
	require.Equal(t, "vid", vm.GetId())
	require.GreaterOrEqual(t, calls, 2)
}
