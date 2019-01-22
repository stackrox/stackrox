package service

import (
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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

// New returns a service object for registering with grpc
func New() Service {
	return &serviceImpl{
		store:         store.Singleton(),
		standardsRepo: standards.RegistrySingleton(),
		clusters:      datastore.Singleton(),
	}
}

type serviceImpl struct {
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

func filterStandards(standards []*v1.ComplianceStandardMetadata, values []string) []string {
	if len(values) == 0 {
		standardIDs := make([]string, 0, len(standards))
		for _, s := range standards {
			standardIDs = append(standardIDs, s.GetId())
		}
		return standardIDs
	}
	var filteredStandards []string
loop:
	for _, standard := range standards {
		standardLower := strings.ToLower(standard.GetName())
		for _, v := range values {
			if strings.HasPrefix(standardLower, strings.ToLower(v)) {
				filteredStandards = append(filteredStandards, standard.GetId())
				continue loop
			}
		}
	}
	return filteredStandards
}

func filterClusters(clusters []*storage.Cluster, values []string) []string {
	if len(values) == 0 {
		clusterIDs := make([]string, 0, len(clusters))
		for _, s := range clusters {
			clusterIDs = append(clusterIDs, s.GetId())
		}
		return clusterIDs
	}
	var filteredClusters []string
loop:
	for _, cluster := range clusters {
		clusterLower := strings.ToLower(cluster.GetName())
		for _, v := range values {
			if strings.HasPrefix(clusterLower, strings.ToLower(v)) {
				filteredClusters = append(filteredClusters, cluster.GetId())
				continue loop
			}
		}
	}
	return filteredClusters
}

func (s *serviceImpl) GetAggregatedResults(ctx context.Context, request *v1.ComplianceAggregation_Request) (*v1.ComplianceAggregation_Response, error) {
	searchMap := search.ParseRawQueryIntoMap(request.GetWhere().GetQuery())

	standards, err := s.standardsRepo.Standards()
	if err != nil {
		return nil, err
	}
	standardIDs := filterStandards(standards, searchMap[search.Standard.String()])

	clusters, err := s.clusters.GetClusters()
	if err != nil {
		return nil, err
	}
	clusterIDs := filterClusters(clusters, searchMap[search.Cluster.String()])

	results, err := s.store.GetLatestRunResultsBatch(clusterIDs, standardIDs)
	if err != nil {
		return nil, err
	}

	return &v1.ComplianceAggregation_Response{
		Results: getAggregatedResults(request.GetGroupBy(), request.GetUnit(), results),
	}, nil
}

func (s *serviceImpl) GetRunResults(ctx context.Context, request *v1.GetComplianceRunResultsRequest) (*v1.GetComplianceRunResultsResponse, error) {
	results, err := s.store.GetLatestRunResults(request.GetClusterId(), request.GetStandardId())
	if err != nil {
		return nil, err
	}
	return &v1.GetComplianceRunResultsResponse{
		Results: results,
	}, nil
}
