package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/convert/storagetov2"
	"github.com/stackrox/rox/central/views/vmcve"
	componentDS "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore"
	cveDS "github.com/stackrox/rox/central/virtualmachine/cve/v2/datastore"
	scanDS "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore"
	vmDS "github.com/stackrox/rox/central/virtualmachine/v2/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	defaultPageSize = 100
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.VirtualMachine)): {
			v2.VirtualMachineV2Service_ListVMs_FullMethodName,
			v2.VirtualMachineV2Service_ListVMCVEs_FullMethodName,
			v2.VirtualMachineV2Service_GetVMDashboardCounts_FullMethodName,
			v2.VirtualMachineV2Service_GetVM_FullMethodName,
			v2.VirtualMachineV2Service_GetVMVulnSummary_FullMethodName,
			v2.VirtualMachineV2Service_ListVMCVEsByVM_FullMethodName,
			v2.VirtualMachineV2Service_GetVMCVEComponents_FullMethodName,
			v2.VirtualMachineV2Service_ListVMComponents_FullMethodName,
			v2.VirtualMachineV2Service_GetVMCVEDetail_FullMethodName,
			v2.VirtualMachineV2Service_ListVMCVEAffectedVMs_FullMethodName,
		},
	})
)

type serviceImpl struct {
	v2.UnimplementedVirtualMachineV2ServiceServer

	vmDS        vmDS.DataStore
	cveDS       cveDS.DataStore
	componentDS componentDS.DataStore
	scanDS      scanDS.DataStore
	cveView     vmcve.CveView
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterVirtualMachineV2ServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterVirtualMachineV2ServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// ListVMs returns a paginated list of VMs with severity counts.
func (s *serviceImpl) ListVMs(ctx context.Context, request *v2.ListVMsRequest) (*v2.ListVMsResponse, error) {
	searchQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}
	paginated.FillPaginationV2(searchQuery, request.GetQuery().GetPagination(), defaultPageSize)

	countQuery := searchQuery.CloneVT()
	countQuery.Pagination = nil
	totalCount, err := s.vmDS.CountVirtualMachines(ctx, countQuery)
	if err != nil {
		return nil, err
	}

	vms, err := s.vmDS.SearchRawVirtualMachines(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	items := make([]*v2.VMListItem, 0, len(vms))
	for _, vm := range vms {
		item := storagetov2.VirtualMachineV2ToListItem(vm)

		// Get severity counts for this VM.
		vmFilter := search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, vm.GetId()).ProtoQuery()
		severityCounts, err := s.cveView.CountBySeverity(ctx, vmFilter)
		if err != nil {
			return nil, err
		}
		item.CveSeverityCounts = storagetov2.SeverityCountsToProto(severityCounts)

		// Get component scan counts for this VM.
		totalComponents, err := s.componentDS.Count(ctx, vmFilter)
		if err != nil {
			return nil, err
		}
		item.ComponentScanCount = &v2.ComponentScanCount{
			Total:   int32(totalComponents),
			Scanned: int32(totalComponents),
		}

		items = append(items, item)
	}

	return &v2.ListVMsResponse{
		Vms:        items,
		TotalCount: int32(totalCount),
	}, nil
}

// ListVMCVEs returns a paginated list of CVEs across all VMs.
func (s *serviceImpl) ListVMCVEs(ctx context.Context, request *v2.ListVMCVEsRequest) (*v2.ListVMCVEsResponse, error) {
	searchQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}
	paginated.FillPaginationV2(searchQuery, request.GetQuery().GetPagination(), defaultPageSize)

	countQuery := searchQuery.CloneVT()
	countQuery.Pagination = nil
	totalCount, err := s.cveView.Count(ctx, countQuery)
	if err != nil {
		return nil, err
	}

	cves, err := s.cveView.Get(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	// Get total VM count for the response.
	totalVMs, err := s.vmDS.CountVirtualMachines(ctx, search.EmptyQuery())
	if err != nil {
		return nil, err
	}

	items := make([]*v2.VMCVEListItem, 0, len(cves))
	for _, cve := range cves {
		item := &v2.VMCVEListItem{
			Cve:              cve.GetCVE(),
			VmSeverityCounts: storagetov2.SeverityCountsToProto(cve.GetVMsBySeverity()),
			TopCvss:          cve.GetTopCVSS(),
			AffectedVmCount:  int32(cve.GetAffectedVMCount()),
			TotalVmCount:     int32(totalVMs),
			EpssProbability:  cve.GetEPSSProbability(),
			PublishedOn:      protocompat.ConvertTimeToTimestampOrNil(cve.GetPublishDate()),
		}
		items = append(items, item)
	}

	return &v2.ListVMCVEsResponse{
		Cves:       items,
		TotalCount: int32(totalCount),
	}, nil
}

// GetVMDashboardCounts returns the total VM and CVE counts for the dashboard.
func (s *serviceImpl) GetVMDashboardCounts(ctx context.Context, request *v2.VMDashboardCountsRequest) (*v2.VMDashboardCountsResponse, error) {
	searchQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}

	vmCount, err := s.vmDS.CountVirtualMachines(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	cveCount, err := s.cveView.Count(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	return &v2.VMDashboardCountsResponse{
		VmCount:  int32(vmCount),
		CveCount: int32(cveCount),
	}, nil
}
