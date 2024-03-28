package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	ruleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	"github.com/stackrox/rox/central/convert/storagetov2"
	v2 "github.com/stackrox/rox/generated/api/v2"
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
			"/v2.ComplianceProfileService/GetComplianceProfile",
			"/v2.ComplianceProfileService/ListComplianceProfiles",
			"/v2.ComplianceProfileService/ListProfileSummaries",
			"/v2.ComplianceProfileService/GetComplianceProfileCount",
		},
	})
)

// New returns a service object for registering with grpc.
func New(complianceRuleDS ruleDS.DataStore) Service {
	return &serviceImpl{
		complianceRuleDS: complianceRuleDS,
	}
}

type serviceImpl struct {
	v2.UnimplementedComplianceProfileServiceServer

	complianceRuleDS ruleDS.DataStore
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

// GetComplianceRuleByName retrieves the specified compliance rule
func (s *serviceImpl) GetComplianceRuleByName(ctx context.Context, req *v2.RuleRequest) (*v2.ComplianceRule, error) {
	if req.GetRuleName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Rule name is required for retrieval")
	}

	rules, err := s.complianceRuleDS.GetRulesByName(ctx, req.GetRuleName())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve compliance rule with name %q.", req.GetRuleName())
	}

	return storagetov2.ComplianceV2Profile(profile), nil
}
