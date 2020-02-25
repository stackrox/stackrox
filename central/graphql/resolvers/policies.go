package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/policyutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()

	utils.Must(
		schema.AddQuery("policies(query: String, pagination: Pagination): [Policy!]!"),
		schema.AddQuery("policy(id: ID): Policy"),
		schema.AddQuery("policyCount(query: String): Int!"),
		schema.AddExtraResolver("Policy", `alerts(query: String, pagination: Pagination): [Alert!]!`),
		schema.AddExtraResolver("Policy", `alertCount(query: String): Int!`),
		schema.AddExtraResolver("Policy", `deployments(query: String, pagination: Pagination): [Deployment!]!`),
		schema.AddExtraResolver("Policy", `deploymentCount(query: String): Int!`),
		schema.AddExtraResolver("Policy", `policyStatus(query: String): String!`),
		schema.AddExtraResolver("Policy", "latestViolation(query: String): Time"),

		schema.AddExtraResolver("PolicyFields", "imageAgeDays: Int!"),
		schema.AddExtraResolver("PolicyFields", "scanAgeDays: Int!"),
		schema.AddExtraResolver("PolicyFields", "noScanExists: Boolean!"),
		schema.AddExtraResolver("PolicyFields", "privileged: Boolean!"),
		schema.AddExtraResolver("PolicyFields", "readOnlyRootFs: Boolean!"),
		schema.AddExtraResolver("PolicyFields", "whitelistEnabled: Boolean!"),
		schema.AddExtraResolver("Policy", `unusedVarSink(query: String): Int`),
	)
}

// Policies returns GraphQL resolvers for all policies
func (resolver *Resolver) Policies(ctx context.Context, args PaginatedQuery) ([]*policyResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Policies")
	if err := readPolicies(ctx); err != nil {
		return nil, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	return resolver.wrapPolicies(resolver.PolicyDataStore.SearchRawPolicies(ctx, q))
}

// Policy returns a GraphQL resolver for a given policy
func (resolver *Resolver) Policy(ctx context.Context, args struct{ *graphql.ID }) (*policyResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Policy")
	if err := readPolicies(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapPolicy(resolver.PolicyDataStore.GetPolicy(ctx, string(*args.ID)))
}

// PolicyCount returns count of all policies across infrastructure
func (resolver *Resolver) PolicyCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "PolicyCount")
	if err := readPolicies(ctx); err != nil {
		return 0, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	results, err := resolver.PolicyDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// Alerts returns GraphQL resolvers for all alerts for this policy
func (resolver *policyResolver) Alerts(ctx context.Context, args PaginatedQuery) ([]*alertResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Policies, "Alerts")
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getRawPolicyQuery())

	return resolver.root.Violations(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

func (resolver *policyResolver) AlertCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Policies, "AlertCount")
	if err := readAlerts(ctx); err != nil {
		return 0, err // could return nil, nil to prevent errors from propagating.
	}

	resolvers, err := resolver.Alerts(ctx, PaginatedQuery{Query: args.Query})
	if err != nil {
		return 0, err
	}

	return int32(len(resolvers)), nil
}

// Deployments returns GraphQL resolvers for all deployments that this policy applies to
func (resolver *policyResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Policies, "Deployments")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}

	if resolver.data.GetDisabled() {
		return nil, nil
	}

	deploymentFilterQuery, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	pagination := deploymentFilterQuery.GetPagination()
	deploymentFilterQuery.Pagination = nil

	deploymentIDs, err := resolver.getDeploymentsForPolicy(ctx)
	if err != nil {
		return nil, err
	}

	deploymentQuery := search.NewConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deploymentIDs...).ProtoQuery(),
		deploymentFilterQuery)
	deploymentQuery.Pagination = pagination

	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapDeployments(deploymentLoader.FromQuery(ctx, deploymentQuery))
}

// DeploymentCount returns the count of all deployments that this policy applies to
func (resolver *policyResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Policies, "DeploymentCount")

	if err := readDeployments(ctx); err != nil {
		return 0, err
	}

	resolvers, err := resolver.Deployments(ctx, PaginatedQuery{Query: args.Query})
	if err != nil {
		return 0, err
	}

	return int32(len(resolvers)), nil
}

// PolicyStatus returns the policy statusof this policy
func (resolver *policyResolver) PolicyStatus(ctx context.Context, args RawQuery) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Policies, "PolicyStatus")
	if resolver.data.GetDisabled() {
		return "", nil
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return "", err
	}

	q, err = search.AddAsConjunction(q, resolver.getPolicyQuery())
	if err != nil {
		return "", err
	}

	activeAlerts, err := anyActiveDeployAlerts(ctx, resolver.root, q)
	if err != nil {
		return "", err
	}

	if activeAlerts {
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

func (resolver *policyResolver) LatestViolation(ctx context.Context, args RawQuery) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Policies, "Latest Violation")

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	q, err = search.AddAsConjunction(q, resolver.getPolicyQuery())
	if err != nil {
		return nil, err
	}

	return getLatestViolationTime(ctx, resolver.root, q)
}

func (resolver *policyResolver) getPolicyQuery() *v1.Query {
	return search.NewQueryBuilder().AddStrings(search.PolicyID, resolver.data.GetId()).ProtoQuery()
}

func (resolver *policyResolver) getRawPolicyQuery() string {
	return search.NewQueryBuilder().AddStrings(search.PolicyID, resolver.data.GetId()).Query()
}

func (resolver *policyResolver) UnusedVarSink(ctx context.Context, args RawQuery) *int32 {
	return nil
}

// Following handle the basic type oneOf fields in policy fields that the codegen does not handle.
///////////////////////////////////////////////////////////////////////////////////////////////////

func (pf *policyFieldsResolver) ImageAgeDays(ctx context.Context) int32 {
	return int32(pf.data.GetImageAgeDays())
}

func (pf *policyFieldsResolver) ScanAgeDays(ctx context.Context) int32 {
	return int32(pf.data.GetScanAgeDays())
}

func (pf *policyFieldsResolver) NoScanExists(ctx context.Context) bool {
	return pf.data.GetNoScanExists()
}

func (pf *policyFieldsResolver) Privileged(ctx context.Context) bool {
	return pf.data.GetPrivileged()
}

func (pf *policyFieldsResolver) WhitelistEnabled(ctx context.Context) bool {
	return pf.data.GetWhitelistEnabled()
}

func (pf *policyFieldsResolver) ReadOnlyRootFs(ctx context.Context) bool {
	return pf.data.GetReadOnlyRootFs()
}
