package service

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/aggregation"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/search"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Compliance)): {
			"/v1.ComplianceService/GetStandards",
			"/v1.ComplianceService/GetStandard",
			"/v1.ComplianceService/GetComplianceControlResults",
			"/v1.ComplianceService/GetComplianceStatistics",
			"/v1.ComplianceService/GetRunResults",
			"/v1.ComplianceService/GetAggregatedResults",
		},
	})
)

// New returns a service object for registering with grpc.
func New(aggregator aggregation.Aggregator, complianceStore store.Store, standardsRepo standards.Repository, clusterStore datastore.DataStore) Service {
	return &serviceImpl{
		aggregator:    aggregator,
		store:         complianceStore,
		standardsRepo: standardsRepo,
		clusters:      clusterStore,
	}
}

type serviceImpl struct {
	aggregator    aggregation.Aggregator
	store         store.Store
	standardsRepo standards.Repository
	clusters      datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterComplianceServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterComplianceServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetStandards returns a list of available standardsRepo
func (s *serviceImpl) GetStandards(context.Context, *v1.Empty) (*v1.GetComplianceStandardsResponse, error) {
	standards, err := s.standardsRepo.Standards()
	if err != nil {
		return nil, err
	}
	return &v1.GetComplianceStandardsResponse{
		Standards: standards,
	}, nil
}

// GetStandard returns details + controls for a given standard
func (s *serviceImpl) GetStandard(ctx context.Context, req *v1.ResourceByID) (*v1.GetComplianceStandardResponse, error) {
	standard, exists, err := s.standardsRepo.Standard(req.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Error(codes.NotFound, req.GetId())
	}
	return &v1.GetComplianceStandardResponse{
		Standard: standard,
	}, nil
}

// GetComplianceControlResults returns controls and evidence
func (s *serviceImpl) GetComplianceControlResults(ctx context.Context, query *v1.RawQuery) (*v1.ComplianceControlResultsResponse, error) {
	q := search.EmptyQuery()
	var err error
	if query.GetQuery() != "" {
		q, err = search.ParseRawQuery(query.GetQuery())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	results, err := s.store.QueryControlResults(q)
	if err != nil {
		return nil, err
	}
	return &v1.ComplianceControlResultsResponse{
		Results: results,
	}, nil
}

func (s *serviceImpl) GetAggregatedResults(ctx context.Context, request *v1.ComplianceAggregation_Request) (*v1.ComplianceAggregation_Response, error) {
	validResults, sources, _, err := s.aggregator.Aggregate(request.GetWhere().GetQuery(), request.GetGroupBy(), request.GetUnit())
	if err != nil {
		return nil, err
	}

	return &v1.ComplianceAggregation_Response{
		Results: validResults,
		Sources: sources,
	}, nil
}

func (s *serviceImpl) GetRunResults(ctx context.Context, request *v1.GetComplianceRunResultsRequest) (*v1.GetComplianceRunResultsResponse, error) {
	results, err := s.store.GetLatestRunResults(request.GetClusterId(), request.GetStandardId())
	if err != nil {
		return nil, err
	}
	return &v1.GetComplianceRunResultsResponse{
		Results:    results.LastSuccessfulResults,
		FailedRuns: results.FailedRuns,
	}, nil
}
