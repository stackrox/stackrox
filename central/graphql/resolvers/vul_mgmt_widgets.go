package resolvers

import (
	"context"
	"sort"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("DeploymentsWithMostSevereViolations", []string{
			"id: ID!",
			"name: String!",
			"namespace: String!",
			"clusterName: String!",
			"failingPolicySeverityCounts: PolicyCounter!",
		}),
		schema.AddQuery("deploymentsWithMostSevereViolations(query: String, pagination: Pagination): [DeploymentsWithMostSevereViolations!]!"),
	)
}

// DeploymentsWithMostSevereViolations returns deployments with their basic info and policies that are failing on it.
func (resolver *Resolver) DeploymentsWithMostSevereViolations(ctx context.Context, args PaginatedQuery) ([]*DeploymentsWithMostSevereViolationsResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "DeploymentsWithMostSevereViolations")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	return deploymentsWithMostSevereViolations(ctx, resolver, q)
}

// DeploymentsWithMostSevereViolationsResolver resolves data about list alert deployments and respective policies failing.
type DeploymentsWithMostSevereViolationsResolver struct {
	root                 *Resolver
	deployment           *storage.ListAlertDeployment
	policySeverityCounts *PolicyCounterResolver
}

// ID returns the deployment ID.
func (r *DeploymentsWithMostSevereViolationsResolver) ID(_ context.Context) graphql.ID {
	return graphql.ID(r.deployment.GetId())
}

// Name returns the deployment name.
func (r *DeploymentsWithMostSevereViolationsResolver) Name(_ context.Context) string {
	return r.deployment.GetName()
}

// Namespace returns the deployment namespace.
func (r *DeploymentsWithMostSevereViolationsResolver) Namespace(_ context.Context) string {
	return r.deployment.GetNamespace()
}

// ClusterName returns the deployment cluster name.
func (r *DeploymentsWithMostSevereViolationsResolver) ClusterName(_ context.Context) string {
	return r.deployment.GetClusterName()
}

// FailingPolicySeverityCounts returns severity bucketed policy counts for policies failing on the deployment.
func (r *DeploymentsWithMostSevereViolationsResolver) FailingPolicySeverityCounts() *PolicyCounterResolver {
	return r.policySeverityCounts
}

// FailingPolicyResolver resolves data about list alert policies
type FailingPolicyResolver struct {
	policy *storage.ListAlertPolicy
}

// ID returns the policy id.
func (r *FailingPolicyResolver) ID(_ context.Context) graphql.ID {
	return graphql.ID(r.policy.GetId())
}

// Severity returns policy severity.
func (r *FailingPolicyResolver) Severity(_ context.Context) string {
	return r.policy.GetSeverity().String()
}

func deploymentsWithMostSevereViolations(ctx context.Context, resolver *Resolver, q *v1.Query) ([]*DeploymentsWithMostSevereViolationsResolver, error) {
	pagination := q.GetPagination()
	q.Pagination = nil

	q, err := search.AddAsConjunction(q, search.NewQueryBuilder().AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery())
	if err != nil {
		return nil, err
	}

	q = paginated.FillDefaultSortOption(q, paginated.GetViolationTimeSortOption())
	alerts, err := resolver.ViolationsDataStore.SearchListAlerts(ctx, q)
	if err != nil {
		return nil, err
	}

	deployments := make(map[string]*storage.ListAlertDeployment)
	deploymentsToFailingPolicies := make(map[string][]*storage.ListAlertPolicy)
	for _, alert := range alerts {
		if _, ok := deployments[alert.GetDeployment().GetId()]; !ok {
			deployments[alert.GetDeployment().GetId()] = alert.GetDeployment()
		}
		deploymentsToFailingPolicies[alert.GetDeployment().GetId()] = append(deploymentsToFailingPolicies[alert.GetDeployment().GetId()], alert.GetPolicy())
	}

	ret := make([]*DeploymentsWithMostSevereViolationsResolver, 0, len(deployments))
	for _, deployment := range deployments {
		ret = append(ret, &DeploymentsWithMostSevereViolationsResolver{
			root:                 resolver,
			deployment:           deployment,
			policySeverityCounts: mapListAlertPoliciesToPolicySeverityCount(deploymentsToFailingPolicies[deployment.GetId()]),
		})
	}

	sortBySeverity(ret)

	return paginate(pagination, ret, nil)
}

func sortBySeverity(deps []*DeploymentsWithMostSevereViolationsResolver) {
	sort.SliceStable(deps, func(i, j int) bool {
		if deps[i].policySeverityCounts.critical != deps[j].policySeverityCounts.critical {
			return deps[i].policySeverityCounts.critical > deps[j].policySeverityCounts.critical
		}

		if deps[i].policySeverityCounts.high != deps[j].policySeverityCounts.high {
			return deps[i].policySeverityCounts.high > deps[j].policySeverityCounts.high
		}

		if deps[i].policySeverityCounts.medium != deps[j].policySeverityCounts.medium {
			return deps[i].policySeverityCounts.medium > deps[j].policySeverityCounts.medium
		}
		return deps[i].policySeverityCounts.low > deps[j].policySeverityCounts.low
	})
}
