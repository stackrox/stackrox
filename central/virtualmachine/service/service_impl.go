package service

import (
	"context"
	"slices"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/convert/storagetov2"
	"github.com/stackrox/rox/central/virtualmachine/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultPageSize = 100
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.VirtualMachine)): {
			v2.VirtualMachineService_GetVirtualMachine_FullMethodName,
			v2.VirtualMachineService_ListVirtualMachines_FullMethodName,
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	v2.UnimplementedVirtualMachineServiceServer
	datastore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterVirtualMachineServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterVirtualMachineServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetVirtualMachine(ctx context.Context, request *v2.GetVirtualMachineRequest) (*v2.VirtualMachine, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id must be specified")
	}

	vm, exists, err := s.datastore.GetVirtualMachine(ctx, request.GetId())

	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "Virtual machine with id %q does not exist", request.GetId())
	}

	return storagetov2.VirtualMachine(vm), nil
}

func (s *serviceImpl) ListVirtualMachines(ctx context.Context, request *v2.ListVirtualMachinesRequest) (*v2.ListVirtualMachinesResponse, error) {
	searchQuery := search.EmptyQuery()
	requestQuery := request.GetQuery().GetQuery()
	if requestQuery != "" {
		parsedQuery, err := search.ParseQuery(requestQuery)
		if err != nil {
			return nil, errors.Wrap(err, "parsing input query")
		}
		searchQuery = parsedQuery
	}
	paginated.FillPaginationV2(searchQuery, request.GetQuery().GetPagination(), defaultPageSize)

	vms, err := s.datastore.SearchRawVirtualMachines(ctx, searchQuery)
	if err != nil {
		// TODO: Handle specific error cases with proper error codes, e.g. duplicate ID
		return nil, err
	}

	v2VMs := make([]*v2.VirtualMachine, 0, len(vms))
	for _, vm := range vms {
		v2VMs = append(v2VMs, storagetov2.VirtualMachine(vm))
	}
	requestQueryPagination := request.GetQuery().GetPagination()
	if requestQueryPagination.GetSortOption() == nil && len(requestQueryPagination.GetSortOptions()) <= 0 {
		// If no sorting is requested, sort by VM name then by VM namespace
		slices.SortFunc(v2VMs, func(vm1, vm2 *v2.VirtualMachine) int {
			if vm1.GetName() < vm2.GetName() {
				return 1
			}
			if vm1.GetName() > vm2.GetName() {
				return -1
			}
			if vm1.GetNamespace() < vm2.GetNamespace() {
				return 1
			}
			if vm1.GetNamespace() > vm2.GetNamespace() {
				return -1
			}
			return 0
		})
	}

	return &v2.ListVirtualMachinesResponse{
		VirtualMachines: v2VMs,
	}, nil
}
