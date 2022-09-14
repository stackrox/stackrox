package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	policyUtils "github.com/stackrox/rox/central/policy/utils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/policyutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()

	utils.Must(
		schema.AddQuery("policies(query: String, pagination: Pagination): [Policy!]!"),
		schema.AddQuery("policy(id: ID): Policy"),
		schema.AddQuery("policyCount(query: String): Int!"),

		schema.AddExtraResolver("Policy", `alertCount(query: String): Int!`),
		schema.AddExtraResolver("Policy", `alerts(query: String, pagination: Pagination): [Alert!]!`),
		schema.AddExtraResolver("Policy", `deploymentCount(query: String): Int!`),
		schema.AddExtraResolver("Policy", `deployments(query: String, pagination: Pagination): [Deployment!]!`),
		schema.AddExtraResolver("Policy", `failingDeploymentCount(query: String): Int!`),
		schema.AddExtraResolver("Policy", `failingDeployments(query: String, pagination: Pagination): [Deployment!]!`),
		schema.AddExtraResolver("Policy", "fullMitreAttackVectors: [MitreAttackVector!]!"),
		schema.AddExtraResolver("Policy", "latestViolation(query: String): Time"),
		schema.AddExtraResolver("Policy", `policyStatus(query: String): String!`),

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
	count, err := resolver.PolicyDataStore.Count(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
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

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getRawPolicyQuery())
	return resolver.root.ViolationCount(ctx, RawQuery{Query: &query})
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

	var err error
	deploymentFilterQuery := search.EmptyQuery()
	if scope, hasScope := scoped.GetScope(ctx); hasScope {
		if field, ok := idField[scope.Level]; ok {
			deploymentFilterQuery = search.NewQueryBuilder().AddExactMatches(field, scope.ID).ProtoQuery()
		}
	} else {
		if deploymentFilterQuery, err = args.AsV1QueryOrEmpty(); err != nil {
			return nil, err
		}

		// If the query contains 'Policy Violated' search field, it means this is a query to find deployments failing
		// on given policy. Since this is a fake search field not belonging to search category, remove the base query
		// that contains the 'Policy Violated' search field and return the rest of the query.
		if filteredQuery, isFailingDeploymentsQuery := inverseFilterFailingDeploymentsQuery(deploymentFilterQuery); isFailingDeploymentsQuery {
			return resolver.failingDeployments(ctx, filteredQuery)
		}
	}

	pagination := deploymentFilterQuery.GetPagination()
	deploymentFilterQuery.Pagination = nil

	deploymentIDs, err := resolver.getDeploymentsForPolicy(ctx)
	if err != nil {
		return nil, err
	}

	deploymentQuery := search.ConjunctionQuery(deploymentFilterQuery,
		search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deploymentIDs...).ProtoQuery())
	deploymentQuery.Pagination = pagination

	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return nil, err
	}
	deploymentResolvers, err := resolver.root.wrapDeployments(deploymentLoader.FromQuery(ctx, deploymentQuery))
	if err != nil {
		return nil, err
	}

	for _, deploymentResolver := range deploymentResolvers {
		deploymentResolver.ctx = scoped.Context(ctx, scoped.Scope{
			Level: v1.SearchCategory_POLICIES,
			ID:    resolver.data.GetId(),
		})
	}
	return deploymentResolvers, nil
}

// FailingDeployments returns GraphQL resolvers for all deployments that this policy is failing on.
func (resolver *policyResolver) FailingDeployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Policies, "FailingDeployments")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}

	if resolver.data.GetDisabled() {
		return nil, nil
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	return resolver.failingDeployments(ctx, q)
}

func (resolver *policyResolver) failingDeployments(ctx context.Context, q *v1.Query) ([]*deploymentResolver, error) {
	alertsQuery := search.ConjunctionQuery(resolver.getPolicyQuery(),
		search.NewQueryBuilder().AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery())
	listAlerts, err := resolver.root.ViolationsDataStore.SearchListAlerts(ctx, alertsQuery)
	if err != nil {
		return nil, err
	}

	deploymentIDs := make([]string, 0, len(listAlerts))
	for _, alert := range listAlerts {
		deploymentIDs = append(deploymentIDs, alert.GetDeployment().GetId())
	}

	deploymentQuery := search.ConjunctionQuery(q, search.NewQueryBuilder().AddDocIDs(deploymentIDs...).ProtoQuery())
	deploymentQuery.Pagination = q.GetPagination()

	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return nil, err
	}
	deploymentResolvers, err := resolver.root.wrapDeployments(deploymentLoader.FromQuery(ctx, deploymentQuery))
	if err != nil {
		return nil, err
	}

	for _, deploymentResolver := range deploymentResolvers {
		deploymentResolver.ctx = scoped.Context(ctx, scoped.Scope{
			Level: v1.SearchCategory_POLICIES,
			ID:    resolver.data.GetId(),
		})
	}
	return deploymentResolvers, nil
}

// DeploymentCount returns the count of all deployments that this policy applies to
func (resolver *policyResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Policies, "DeploymentCount")

	if err := readDeployments(ctx); err != nil {
		return 0, err
	}

	return resolver.DeploymentCount(ctx, RawQuery{Query: args.Query})
}

// FailingDeploymentCount returns the count of deployments that this policy is failing on
func (resolver *policyResolver) FailingDeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Policies, "DeploymentCount")
	if err := readAlerts(ctx); err != nil {
		return 0, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	q = search.ConjunctionQuery(q, resolver.getPolicyQuery(),
		search.NewQueryBuilder().AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery())
	count, err := resolver.root.ViolationsDataStore.Count(ctx, q)
	if err != nil {
		return 0, err
	}
	// This is because alert <-> policy <-> deployment is generally 1:1.
	return int32(count), nil
}

// PolicyStatus returns the policy statusof this policy
func (resolver *policyResolver) PolicyStatus(ctx context.Context, args RawQuery) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Policies, "PolicyStatus")
	if resolver.data.GetDisabled() {
		return "", nil
	}

	var err error
	q := search.EmptyQuery()
	if scope, hasScope := scoped.GetScope(resolver.ctx); hasScope {
		if field, ok := idField[scope.Level]; ok {
			q = search.NewQueryBuilder().AddExactMatches(field, scope.ID).ProtoQuery()
		}
	} else {
		if q, err = args.AsV1QueryOrEmpty(); err != nil {
			return "", err
		}
	}

	q, err = search.AddAsConjunction(q, resolver.getPolicyQuery())
	if err != nil {
		return "", err
	}

	activeAlerts, err := deployTimeAlertExists(ctx, resolver.root, q)
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

	deploymentExclusionQuery := policyutils.DeploymentExclusionToQuery(resolver.data.GetExclusions())
	exclusionResults, err := resolver.root.DeploymentDataStore.Search(ctx, deploymentExclusionQuery)
	if err != nil {
		return nil, err
	}

	return search.ResultsToIDSet(scopeQueryResults).
		Difference(search.ResultsToIDSet(exclusionResults)).AsSlice(), nil
}

func (resolver *policyResolver) LatestViolation(ctx context.Context, args RawQuery) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Policies, "Latest Violation")

	var err error
	q := search.EmptyQuery()
	if scope, hasScope := scoped.GetScope(resolver.ctx); hasScope {
		if field, ok := idField[scope.Level]; ok {
			q = search.NewQueryBuilder().AddExactMatches(field, scope.ID).ProtoQuery()
		}
	} else {
		if q, err = args.AsV1QueryOrEmpty(); err != nil {
			return nil, err
		}
	}

	q, err = search.AddAsConjunction(q, resolver.getPolicyQuery())
	if err != nil {
		return nil, err
	}

	return getLatestViolationTime(ctx, resolver.root, q)
}

func (resolver *policyResolver) FullMitreAttackVectors(ctx context.Context) ([]*mitreAttackVectorResolver, error) {
	return resolver.root.wrapMitreAttackVectors(
		policyUtils.GetFullMitreAttackVectors(resolver.root.mitreStore, resolver.data),
	)
}

func (resolver *policyResolver) getPolicyQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.PolicyID, resolver.data.GetId()).ProtoQuery()
}

func (resolver *policyResolver) getRawPolicyQuery() string {
	return search.NewQueryBuilder().AddExactMatches(search.PolicyID, resolver.data.GetId()).Query()
}

func (resolver *policyResolver) UnusedVarSink(ctx context.Context, args RawQuery) *int32 {
	return nil
}

func inverseFilterFailingDeploymentsQuery(q *v1.Query) (*v1.Query, bool) {
	failingDeploymentsQuery := false
	local := q.Clone()
	filtered, _ := search.FilterQuery(local, func(bq *v1.BaseQuery) bool {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok {
			if matchFieldQuery.MatchFieldQuery.GetField() == search.PolicyViolated.String() {
				failingDeploymentsQuery = true
				return false
			}
		}
		return true
	})

	return filtered, failingDeploymentsQuery
}
