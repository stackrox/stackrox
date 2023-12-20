package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	"github.com/stackrox/rox/central/convert/storagetov2"
	v2 "github.com/stackrox/rox/generated/api/v2"
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
	maxPaginationLimit = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Compliance)): {
			"/v2.ComplianceProfileService/GetComplianceProfile",
			"/v2.ComplianceProfileService/ListComplianceProfiles",
			"/v2.ComplianceProfileService/GetComplianceProfileCount",
		},
	})
)

// New returns a service object for registering with grpc.
func New(complianceProfilesDS profileDS.DataStore) Service {
	return &serviceImpl{
		complianceProfilesDS: complianceProfilesDS,
	}
}

type serviceImpl struct {
	v2.UnimplementedComplianceProfileServiceServer

	complianceProfilesDS profileDS.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterComplianceProfileServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterComplianceProfileServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetComplianceProfile retrieves the specified compliance profile
func (s *serviceImpl) GetComplianceProfile(ctx context.Context, req *v2.ResourceByID) (*v2.ComplianceProfile, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Scan configuration name is required for retrieval")
	}

	profile, found, err := s.complianceProfilesDS.GetProfile(ctx, req.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve compliance profile with id %q.", req.GetId())
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "compliance profile with id %q does not exist", req.GetId())
	}

	return storagetov2.ComplianceV2Profile(profile), nil
}

// ListComplianceProfiles returns profiles matching given query
func (s *serviceImpl) ListComplianceProfiles(ctx context.Context, query *v2.RawQuery) (*v2.ListComplianceProfilesResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	profiles, err := s.complianceProfilesDS.SearchProfiles(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance profiles for query %v", query)
	}

	return &v2.ListComplianceProfilesResponse{
		Profiles: storagetov2.ComplianceV2Profiles(profiles),
	}, nil
}

// GetComplianceProfileCount returns counts of profiles matching query
func (s *serviceImpl) GetComplianceProfileCount(ctx context.Context, request *v2.RawQuery) (*v2.CountComplianceProfilesResponse, error) {
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	profileCount, err := s.complianceProfilesDS.CountProfiles(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrap(errox.NotFound, err.Error())
	}
	return &v2.CountComplianceProfilesResponse{
		Count: int32(profileCount),
	}, nil
}
