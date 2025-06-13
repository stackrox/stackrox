package service

import (
	"context"
	"math"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/central/virtualmachine/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	maxVirtualMachinesReturned = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.VirtualMachine)): {
			v1.VirtualMachineService_GetVirtualMachine_FullMethodName,
			v1.VirtualMachineService_ListVirtualMachines_FullMethodName,
		},
		user.With(permissions.Modify(resources.VirtualMachine)): {
			v1.VirtualMachineService_DeleteVirtualMachines_FullMethodName,
			v1.VirtualMachineService_DeleteVirtualMachine_FullMethodName,
			v1.VirtualMachineService_CreateVirtualMachine_FullMethodName,
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	v1.UnimplementedVirtualMachineServiceServer
	datastore        datastore.DataStore
	clusterSACHelper sachelper.ClusterSacHelper
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterVirtualMachineServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterVirtualMachineServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) CreateVirtualMachine(ctx context.Context, request *v1.CreateVirtualMachineRequest) (*storage.VirtualMachine, error) {
	println(request)
	if request != nil {
		println(request.VirtualMachine)
	}
	if request == nil || request.VirtualMachine.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id must be specified")
	}

	if err := s.datastore.UpsertVirtualMachine(ctx, request.VirtualMachine); err != nil {
		return nil, err
	}

	return request.VirtualMachine, nil
}

func (s *serviceImpl) GetVirtualMachine(ctx context.Context, request *v1.GetVirtualMachineRequest) (*storage.VirtualMachine, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id must be specified")
	}

	vm, exists, err := s.datastore.GetVirtualMachine(ctx, request.GetId())

	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "Virtual machine with id %q does not exist", request.GetId())
	}

	return vm, nil
}

func (s *serviceImpl) ListVirtualMachines(ctx context.Context, request *v1.RawQuery) (*v1.ListVirtualMachinesResponse, error) {
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	// Fill in pagination.
	paginated.FillPagination(parsedQuery, request.GetPagination(), maxVirtualMachinesReturned)

	vms, err := s.datastore.SearchRawVirtualMachines(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}

	return &v1.ListVirtualMachinesResponse{
		VirtualMachines: vms,
	}, nil
}

func (s *serviceImpl) DeleteVirtualMachine(ctx context.Context, request *v1.DeleteVirtualMachineRequest) (*v1.DeleteVirtualMachineResponse, error) {
	response := v1.DeleteVirtualMachineResponse{}
	if request.Id == "" {
		return &response, errors.New("id cannot be empty")
	}

	if err := s.datastore.DeleteVirtualMachines(ctx, request.Id); err != nil {
		return &response, err
	} else {
		response.Success = true
		return &response, nil
	}
}

func (s *serviceImpl) DeleteVirtualMachines(ctx context.Context, request *v1.DeleteVirtualMachinesRequest) (*v1.DeleteVirtualMachinesResponse, error) {
	if request.GetQuery() == nil {
		return nil, errors.New("a scoping query is required")
	}

	query, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "error parsing query: %v", err)
	}
	paginated.FillPagination(query, request.GetQuery().GetPagination(), math.MaxInt32)

	results, err := s.datastore.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	response := &v1.DeleteVirtualMachinesResponse{
		NumDeleted: uint32(len(results)),
		DryRun:     !request.GetConfirm(),
	}

	if !request.GetConfirm() {
		return response, nil
	}

	idSlice := search.ResultsToIDs(results)
	if err := s.datastore.DeleteVirtualMachines(ctx, idSlice...); err != nil {
		return nil, err
	}
	return response, nil
}
