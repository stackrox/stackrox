package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/aggregation"
	complianceDS "github.com/stackrox/rox/central/compliance/datastore"
	complianceDSTypes "github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/complianceoperator/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Compliance)): {
			"/v1.ComplianceService/GetStandards",
			"/v1.ComplianceService/GetStandard",
			"/v1.ComplianceService/GetComplianceStatistics",
			"/v1.ComplianceService/GetRunResults",
			"/v1.ComplianceService/GetAggregatedResults",
		},
		user.With(permissions.Modify(resources.Compliance)): {
			"/v1.ComplianceService/UpdateComplianceStandardConfig",
		},
	})
)

// New returns a service object for registering with grpc.
func New(aggregator aggregation.Aggregator, complianceDataStore complianceDS.DataStore, standardsRepo standards.Repository, clusterStore datastore.DataStore, manager manager.Manager) Service {
	return &serviceImpl{
		aggregator:          aggregator,
		complianceDataStore: complianceDataStore,
		standardsRepo:       standardsRepo,
		clusters:            clusterStore,
		manager:             manager,
	}
}

type serviceImpl struct {
	v1.UnimplementedComplianceServiceServer

	aggregator          aggregation.Aggregator
	complianceDataStore complianceDS.DataStore
	standardsRepo       standards.Repository
	clusters            datastore.DataStore
	manager             manager.Manager
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
func (s *serviceImpl) GetStandards(ctx context.Context, _ *v1.Empty) (*v1.GetComplianceStandardsResponse, error) {
	standards, err := s.standardsRepo.Standards()

	if err != nil {
		return nil, err
	}
	// Filter standards by active
	filteredStandards := standards[:0]
	for _, standard := range standards {
		hide, exists, _ := s.complianceDataStore.GetConfig(ctx, standard.GetId())
		if s.manager.IsStandardActive(standard.GetId()) {
			if exists {
				standard.HideScanResults = hide.HideScanResults
			}
			filteredStandards = append(filteredStandards, standard)
		}
	}

	return &v1.GetComplianceStandardsResponse{
		Standards: filteredStandards,
	}, nil
}

// GetStandard returns details + controls for a given standard
func (s *serviceImpl) GetStandard(ctx context.Context, req *v1.ResourceByID) (*v1.GetComplianceStandardResponse, error) {
	standard, exists, err := s.standardsRepo.Standard(req.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrap(errox.NotFound, req.GetId())
	}

	hide, exists, _ := s.complianceDataStore.GetConfig(ctx, req.GetId())
	if exists {
		standard.Metadata.HideScanResults = hide.HideScanResults
	}
	return &v1.GetComplianceStandardResponse{
		Standard: standard,
	}, nil
}

func (s *serviceImpl) GetAggregatedResults(ctx context.Context, request *v1.ComplianceAggregationRequest) (*storage.ComplianceAggregation_Response, error) {
	if request.GetUnit() == storage.ComplianceAggregation_UNKNOWN {
		request.Unit = storage.ComplianceAggregation_CHECK
	}
	validResults, sources, _, err := s.aggregator.Aggregate(ctx, request.GetWhere().GetQuery(), request.GetGroupBy(), request.GetUnit())
	if err != nil {
		return nil, err
	}

	filteredSources := sources[:0]
	for _, source := range sources {
		config, exists, err := s.complianceDataStore.GetConfig(ctx, source.GetStandardId())
		if err != nil {
			return nil, err
		}
		// Not exists implies standards is not hidden
		if exists && config.GetHideScanResults() {
			continue
		}
		filteredSources = append(filteredSources, source)
	}
	return &storage.ComplianceAggregation_Response{
		Results: validResults,
		Sources: filteredSources,
	}, nil
}

func (s *serviceImpl) GetRunResults(ctx context.Context, request *v1.GetComplianceRunResultsRequest) (*v1.GetComplianceRunResultsResponse, error) {
	var results complianceDSTypes.ResultsWithStatus
	var err error
	if request.GetRunId() != "" {
		results, err = s.complianceDataStore.GetSpecificRunResults(ctx, request.GetClusterId(), request.GetStandardId(), request.GetRunId(), complianceDSTypes.WithMessageStrings)
	} else {
		results, err = s.complianceDataStore.GetLatestRunResults(ctx, request.GetClusterId(), request.GetStandardId(), complianceDSTypes.WithMessageStrings)
	}
	if err != nil {
		return nil, err
	}
	return &v1.GetComplianceRunResultsResponse{
		Results:    results.LastSuccessfulResults,
		FailedRuns: results.FailedRuns,
	}, nil
}

// UpdateComplianceStandardConfig updates compliance standards config
func (s *serviceImpl) UpdateComplianceStandardConfig(ctx context.Context, req *v1.UpdateComplianceRequest) (*v1.Empty, error) {
	_, exists, err := s.standardsRepo.Standard(req.GetId())

	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrap(errox.NotFound, req.GetId())
	}
	err = s.complianceDataStore.UpdateConfig(ctx, req.GetId(), req.GetHideScanResults())
	if err != nil {
		return nil, err
	}

	return &v1.Empty{}, nil
}
