package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/policyutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()

	utils.Must(
		schema.AddQuery("policies(query: String): [Policy!]!"),
		schema.AddQuery("policy(id: ID): Policy"),
		schema.AddExtraResolver("Policy", `alerts: [Alert!]!`),
		schema.AddExtraResolver("Policy", `alertCount: Int!`),
		schema.AddExtraResolver("Policy", `deployments(query: String): [Deployment!]!`),
		schema.AddExtraResolver("Policy", `deploymentCount: Int!`),
		schema.AddExtraResolver("Policy", `policyStatus: String!`),
	)
}

// Policies returns GraphQL resolvers for all policies
func (resolver *Resolver) Policies(ctx context.Context, args rawQuery) ([]*policyResolver, error) {
	if err := readPolicies(ctx); err != nil {
		return nil, err
	}
	q, err := args.AsV1Query()
	if err != nil {
		return nil, err
	}
	if q == nil {
		return resolver.wrapPolicies(resolver.PolicyDataStore.GetPolicies(ctx))
	}
	return resolver.wrapPolicies(resolver.PolicyDataStore.SearchRawPolicies(ctx, q))
}

// Policy returns a GraphQL resolver for a given policy
func (resolver *Resolver) Policy(ctx context.Context, args struct{ *graphql.ID }) (*policyResolver, error) {
	if err := readPolicies(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapPolicy(resolver.PolicyDataStore.GetPolicy(ctx, string(*args.ID)))
}

// Alerts returns GraphQL resolvers for all alerts for this policy
func (resolver *policyResolver) Alerts(ctx context.Context) ([]*alertResolver, error) {
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddStrings(search.PolicyID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapAlerts(
		resolver.root.ViolationsDataStore.SearchRawAlerts(ctx, query))
}

func (resolver *policyResolver) AlertCount(ctx context.Context) (int32, error) {
	if err := readAlerts(ctx); err != nil {
		return 0, err // could return nil, nil to prevent errors from propagating.
	}
	query := search.NewQueryBuilder().AddStrings(search.PolicyID, resolver.data.GetId()).ProtoQuery()
	results, err := resolver.root.ViolationsDataStore.Search(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// Deployments returns GraphQL resolvers for all deployments that this policy applies to
func (resolver *policyResolver) Deployments(ctx context.Context, args rawQuery) ([]*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}

	deploymentFilterQuery, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	deploymentIDs, err := resolver.getDeploymentsForPolicy(ctx)
	if err != nil {
		return nil, err
	}
	deploymentIDQuery := search.NewQueryBuilder().AddDocIDs(deploymentIDs...).ProtoQuery()

	deployments, err := resolver.root.DeploymentDataStore.SearchRawDeployments(ctx,
		search.NewConjunctionQuery(deploymentIDQuery, deploymentFilterQuery))
	return resolver.root.wrapDeployments(deployments, err)
}

// DeploymentCount returns the count of all deployments that this policy applies to
func (resolver *policyResolver) DeploymentCount(ctx context.Context) (int32, error) {
	if err := readDeployments(ctx); err != nil {
		return 0, err
	}

	deploymentIDs, err := resolver.getDeploymentsForPolicy(ctx)
	if err != nil {
		return 0, err
	}
	return int32(len(deploymentIDs)), nil
}

// PolicyStatus returns the policy statusof this policy
func (resolver *policyResolver) PolicyStatus(ctx context.Context) (string, error) {
	alertsActive, err := resolver.anyActiveDeployAlerts(ctx)

	if err != nil {
		return "", err
	}

	if alertsActive {
		return "fail", nil
	}

	return "pass", nil
}

func (resolver *policyResolver) getDeploymentsForPolicy(ctx context.Context) ([]string, error) {
	scopeQuery := policyutils.ScopeToQuery(resolver.data.GetScope())
	scopeQueryResults, err := resolver.root.DeploymentDataStore.Search(ctx, scopeQuery)
	if err != nil {
		return nil, err
	}

	deploymentWhitelistQuery := policyutils.DeploymentWhitelistToQuery(resolver.data.GetWhitelists())
	whitelistResults, err := resolver.root.DeploymentDataStore.Search(ctx, deploymentWhitelistQuery)
	if err != nil {
		return nil, err
	}

	return search.ResultsToIDSet(scopeQueryResults).
		Difference(search.ResultsToIDSet(whitelistResults)).AsSlice(), nil
}

func (resolver *policyResolver) anyActiveDeployAlerts(ctx context.Context) (bool, error) {
	if err := readAlerts(ctx); err != nil {
		return false, err
	}

	policy := resolver.data

	q := search.NewQueryBuilder().AddExactMatches(search.PolicyID, policy.GetId()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddStrings(search.LifecycleStage, storage.LifecycleStage_DEPLOY.String()).
		ProtoQuery()
	q.Pagination = &v1.QueryPagination{
		Limit: 1,
	}

	results, err := resolver.root.ViolationsDataStore.Search(ctx, q)
	return len(results) != 0, err
}
