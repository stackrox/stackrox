package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
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
			v2.ComplianceRuleService_GetComplianceRule_FullMethodName,
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
	v2.UnimplementedComplianceRuleServiceServer

	complianceRuleDS ruleDS.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterComplianceRuleServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterComplianceRuleServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetComplianceRule retrieves the specified compliance rule
func (s *serviceImpl) GetComplianceRule(ctx context.Context, req *v2.RuleRequest) (*v2.ComplianceRule, error) {
	if req.GetRuleName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Rule name is required for retrieval")
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the scan config name as an exact match
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleName, req.GetRuleName()).ProtoQuery(),
		parsedQuery,
	)

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, req.GetQuery().GetPagination(), maxPaginationLimit)

	rules, err := s.complianceRuleDS.SearchRules(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve compliance rule with name %q.", req.GetRuleName())
	}

	if len(rules) == 0 {
		return nil, nil
	}

	return storagetov2.ComplianceRule(rules[0]), nil
}
