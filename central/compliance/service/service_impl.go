package service

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/compliance/datastore"
	"github.com/stackrox/rox/central/compliance/standards"
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
		},
	})
)

// New returns a service object for registering with grpc
func New() Service {
	return &serviceImpl{
		store:     datastore.Fake(),
		standards: standards.Fake(),
	}
}

type serviceImpl struct {
	store     datastore.DataStore
	standards standards.Standards
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

// GetStandards returns a list of available standards
func (s *serviceImpl) GetStandards(context.Context, *v1.Empty) (*v1.GetComplianceStandardsResponse, error) {
	standards, err := s.standards.Standards()
	if err != nil {
		return nil, err
	}
	return &v1.GetComplianceStandardsResponse{
		Standards: standards,
	}, nil
}

// GetStandard returns details + controls for a given standard
func (s *serviceImpl) GetStandard(ctx context.Context, req *v1.ResourceByID) (*v1.GetComplianceStandardResponse, error) {
	metadata, exists, err := s.standards.Standard(req.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Error(codes.NotFound, req.GetId())
	}
	controls, err := s.standards.Controls(req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.GetComplianceStandardResponse{
		Standard: &v1.ComplianceStandard{
			Metadata: metadata,
			Controls: controls,
		},
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
