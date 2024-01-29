package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/namespace"
	"github.com/stackrox/rox/central/policy/matcher"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddExtraResolvers("Namespace", []string{
			"cluster: Cluster!",
			"complianceResults(query: String): [ControlResult!]!",
			"deploymentCount(query: String): Int!",
			"deployments(query: String, pagination: Pagination): [Deployment!]!",
			"failingPolicyCounter(query: String): PolicyCounter",
			"imageComponentCount(query: String): Int!",
			"imageComponents(query: String, pagination: Pagination): [ImageComponent!]!",
			"imageCount(query: String): Int!",
			"images(query: String, pagination: Pagination): [Image!]!",
			"imageVulnerabilityCount(query: String): Int!",
			"imageVulnerabilityCounter(query: String): VulnerabilityCounter!",
			"imageVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [ImageVulnerability!]!",
			"k8sRoleCount(query: String): Int!",
			"k8sRoles(query: String, pagination: Pagination): [K8SRole!]!",
			"latestViolation(query: String): Time",
			"networkPolicyCount(query: String): Int!",
			"plottedImageVulnerabilities(query: String): PlottedImageVulnerabilities!",
			"policies(query: String, pagination: Pagination): [Policy!]!",
			"policyCount(query: String): Int!",
			"policyStatus(query: String): PolicyStatus!",
			"policyStatusOnly(query: String): String!",
			"subjectCount(query: String): Int!",
			"subjects(query: String, pagination: Pagination): [Subject!]!",
			"secretCount(query: String): Int!",
			"secrets(query: String, pagination: Pagination): [Secret!]!",
			"serviceAccountCount(query: String): Int!",
			"serviceAccounts(query: String, pagination: Pagination): [ServiceAccount!]!",
			"unusedVarSink(query: String): Int",
			"risk: Risk",
		}),
		// NOTE: This will not populate numDeployments, numNetworkPolicies, or numSecrets in Namespace! Use sub-resolvers for that.
		schema.AddQuery("namespace(id: ID!): Namespace"),
		schema.AddQuery("namespaceByClusterIDAndName(clusterID: ID!, name: String!): Namespace"),
		schema.AddQuery("namespaceCount(query: String): Int!"),
		// NOTE: This will not populate numDeployments, numNetworkPolicies, or numSecrets in Namespace! Use sub-resolvers for that.
		schema.AddQuery("namespaces(query: String, pagination: Pagination): [Namespace!]!"),
	)
}

func (resolver *namespaceResolver) getClusterNamespaceRawQuery() string {
	return search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.Metadata.GetName()).
		Query()
}

func (resolver *namespaceResolver) getClusterNamespaceQuery() *v1.Query {
	return search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.Metadata.GetName()).
		ProtoQuery()
}

func (resolver *namespaceResolver) getNamespaceConjunctionQuery(args RawQuery) (*v1.Query, error) {
	q1 := resolver.getClusterNamespaceQuery()
	if args.String() == "" {
		return q1, nil
	}

	q2, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	return search.AddAsConjunction(q2, q1)
}

// Namespace returns a GraphQL resolver for the given namespace.
func (resolver *Resolver) Namespace(ctx context.Context, args struct{ graphql.ID }) (*namespaceResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Namespace")
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	ns, ok, err := namespace.ResolveMetadataOnlyByID(ctx, string(args.ID), resolver.NamespaceDataStore)
	return resolver.wrapNamespaceWithContext(ctx, ns, ok, err)
}

// Namespaces returns GraphQL resolvers for all namespaces based on an optional query.
func (resolver *Resolver) Namespaces(ctx context.Context, args PaginatedQuery) ([]*namespaceResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Namespaces")
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	nsLoader, err := loaders.GetNamespaceLoader(ctx)
	if err != nil {
		return nil, err
	}
	metadataSlice, err := nsLoader.FromQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	namespaces := make([]*v1.Namespace, 0, len(metadataSlice))
	for _, metadata := range metadataSlice {
		namespaces = append(namespaces, &v1.Namespace{Metadata: metadata})
	}
	return resolver.wrapNamespacesWithContext(ctx, namespaces, nil)
}

type clusterIDAndNameQuery struct {
	ClusterID graphql.ID
	Name      string
}

// NamespaceByClusterIDAndName returns a GraphQL resolver for the (unique) namespace specified by this query.
func (resolver *Resolver) NamespaceByClusterIDAndName(ctx context.Context, args clusterIDAndNameQuery) (*namespaceResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "NamespaceByClusterIDAndName")
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}

	ns, ok, err := namespace.ResolveByClusterIDAndName(ctx, string(args.ClusterID), args.Name, resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore)
	return resolver.wrapNamespaceWithContext(ctx, ns, ok, err)
}

// NamespaceCount returns count of all clusters across infrastructure
func (resolver *Resolver) NamespaceCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NamespaceCount")
	if err := readNamespaces(ctx); err != nil {
		return 0, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	count, err := resolver.NamespaceDataStore.Count(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

func (resolver *namespaceResolver) ComplianceResults(ctx context.Context, args RawQuery) ([]*controlResultResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "ComplianceResults")
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}

	runResults, err := resolver.root.ComplianceAggregator.GetResultsWithEvidence(ctx, args.String())
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	nsID := resolver.data.GetMetadata().GetId()
	output.addDeploymentData(resolver.root, runResults, func(d *storage.ComplianceDomain_Deployment, _ *v1.ComplianceControl) bool {
		return d.GetNamespaceId() == nsID
	})

	return *output, nil
}

// SubjectCount returns the count of Subjects which have any permission on this namespace or the cluster it belongs to
func (resolver *namespaceResolver) SubjectCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "SubjectCount")
	if err := readK8sSubjects(ctx); err != nil {
		return 0, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return 0, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	subjects, err := resolver.getSubjects(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(subjects)), nil
}

// Subjects returns the Subjects which have any permission in namespace or cluster wide
func (resolver *namespaceResolver) Subjects(ctx context.Context, args PaginatedQuery) ([]*subjectResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Subjects")
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}
	var resolvers []*subjectResolver
	baseQuery, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	pagination := baseQuery.GetPagination()
	baseQuery.Pagination = nil

	subjects, err := resolver.getSubjects(ctx, baseQuery)
	if err != nil {
		return nil, err
	}
	for _, subject := range subjects {
		resolvers = append(resolvers, &subjectResolver{ctx, resolver.root, subject})
	}

	return paginate(pagination, resolvers, nil)
}

// ServiceAccountCount returns the count of ServiceAccounts which have any permission on this cluster namespace
func (resolver *namespaceResolver) ServiceAccountCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "ServiceAccountCount")
	if err := readServiceAccounts(ctx); err != nil {
		return 0, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	q, err = search.AddAsConjunction(resolver.getClusterNamespaceQuery(), q)
	if err != nil {
		return 0, err
	}

	count, err := resolver.root.ServiceAccountsDataStore.Count(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// ServiceAccounts returns the ServiceAccounts which have any permission on this cluster namespace
func (resolver *namespaceResolver) ServiceAccounts(ctx context.Context, args PaginatedQuery) ([]*serviceAccountResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "ServiceAccounts")
	if err := readServiceAccounts(ctx); err != nil {
		return nil, err
	}

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getClusterNamespaceRawQuery())

	return resolver.root.ServiceAccounts(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

// K8sRoleCount returns count of K8s roles in this cluster namespace
func (resolver *namespaceResolver) K8sRoleCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "K8sRoleCount")
	if err := readK8sRoles(ctx); err != nil {
		return 0, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	q, err = search.AddAsConjunction(resolver.getClusterNamespaceQuery(), q)
	if err != nil {
		return 0, err
	}

	count, err := resolver.root.K8sRoleStore.Count(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// K8sRoles returns count of K8s roles in this cluster namespace
func (resolver *namespaceResolver) K8sRoles(ctx context.Context, args PaginatedQuery) ([]*k8SRoleResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "K8sRoles")
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getClusterNamespaceRawQuery())

	return resolver.root.K8sRoles(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

func (resolver *namespaceResolver) Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Images")
	return resolver.root.Images(resolver.namespaceScopeContext(ctx), args)
}

func (resolver *namespaceResolver) ImageCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "ImageCount")
	return resolver.root.ImageCount(resolver.namespaceScopeContext(ctx), args)
}

func (resolver *namespaceResolver) getApplicablePolicies(ctx context.Context, q *v1.Query) ([]*storage.Policy, error) {
	policyLoader, err := loaders.GetPolicyLoader(ctx)
	if err != nil {
		return nil, err
	}

	policies, err := policyLoader.FromQuery(ctx, q)
	if err != nil {
		return nil, err
	}

	applicable, _ := matcher.NewNamespaceMatcher(resolver.data.Metadata).FilterApplicablePolicies(policies)
	return applicable, nil
}

// PolicyCount returns count of policies applicable to this namespace
func (resolver *namespaceResolver) PolicyCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "PolicyCount")

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	policies, err := resolver.getApplicablePolicies(ctx, query)
	if err != nil {
		return 0, err
	}

	return int32(len(policies)), nil
}

// Policies returns all the policies applicable to this namespace
func (resolver *namespaceResolver) Policies(ctx context.Context, args PaginatedQuery) ([]*policyResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Policies")

	if err := readWorkflowAdministration(ctx); err != nil {
		return nil, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	// remove pagination from query since we want to paginate the final result
	pagination := q.GetPagination()
	q.Pagination = &v1.QueryPagination{
		SortOptions: pagination.GetSortOptions(),
	}

	policyResolvers, err := resolver.root.wrapPolicies(resolver.getApplicablePolicies(ctx, q))
	if err != nil {
		return nil, err
	}
	for _, policyResolver := range policyResolvers {
		policyResolver.ctx = scoped.Context(ctx, scoped.Scope{
			Level: v1.SearchCategory_NAMESPACES,
			ID:    resolver.data.GetMetadata().GetId(),
		})
	}

	return paginate(pagination, policyResolvers, nil)
}

// FailingPolicyCounter returns a policy counter for all the failed policies.
func (resolver *namespaceResolver) FailingPolicyCounter(ctx context.Context, args RawQuery) (*PolicyCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "FailingPolicyCounter")
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	q, err = search.AddAsConjunction(q, resolver.getClusterNamespaceQuery())
	if err != nil {
		return nil, err
	}

	alerts, err := resolver.root.ViolationsDataStore.SearchListAlerts(ctx, q)
	if err != nil {
		return nil, nil
	}
	return mapListAlertsToPolicySeverityCount(alerts), nil
}

// PolicyStatus returns true if there is no policy violation for this namespace
func (resolver *namespaceResolver) PolicyStatus(ctx context.Context, args RawQuery) (*policyStatusResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "PolicyStatus")

	if err := readAlerts(ctx); err != nil {
		return nil, err
	}

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	alerts, err := resolver.getActiveDeployAlerts(ctx, query)
	if err != nil {
		return nil, err
	}

	scopedCtx := scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_NAMESPACES,
		ID:    resolver.data.GetMetadata().GetId(),
	})

	if len(alerts) == 0 {
		return &policyStatusResolver{scopedCtx, resolver.root, "pass", nil}, nil
	}

	policyIDs := set.NewStringSet()
	for _, alert := range alerts {
		policyIDs.Add(alert.GetPolicy().GetId())
	}

	return &policyStatusResolver{scopedCtx, resolver.root, "fail", policyIDs.AsSlice()}, nil
}

// PolicyStatusOnly returns 'fail' if there are policy violations for this namespace
func (resolver *namespaceResolver) PolicyStatusOnly(ctx context.Context, args RawQuery) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "PolicyStatusOnly")
	if err := readAlerts(ctx); err != nil {
		return "", err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return "", err
	}

	q.Pagination = &v1.QueryPagination{
		Limit: 1,
	}
	results, err := resolver.root.ViolationsDataStore.Search(ctx,
		search.ConjunctionQuery(q,
			search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
				AddExactMatches(search.Namespace, resolver.data.GetMetadata().GetName()).
				AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()))
	if err != nil {
		return "", err
	}

	if len(results) > 0 {
		return "fail", nil
	}
	return "pass", nil
}

func (resolver *namespaceResolver) getActiveDeployAlerts(ctx context.Context, q *v1.Query) ([]*storage.ListAlert, error) {
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}

	namespace := resolver.data
	q = search.ConjunctionQuery(q,
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, namespace.GetMetadata().GetClusterId()).
			AddExactMatches(search.Namespace, namespace.GetMetadata().GetName()).
			AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).
			AddExactMatches(search.LifecycleStage, storage.LifecycleStage_DEPLOY.String()).ProtoQuery())
	q = paginated.FillDefaultSortOption(q, paginated.GetViolationTimeSortOption())

	return resolver.root.ViolationsDataStore.SearchListAlerts(ctx, q)
}

func (resolver *namespaceResolver) ImageComponents(ctx context.Context, args PaginatedQuery) ([]ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "ImageComponents")
	return resolver.root.ImageComponents(resolver.namespaceScopeContext(ctx), args)
}

func (resolver *namespaceResolver) ImageComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "ImageComponentCount")
	return resolver.root.ImageComponentCount(resolver.namespaceScopeContext(ctx), args)
}

func (resolver *namespaceResolver) namespaceScopeContext(ctx context.Context) context.Context {
	if ctx == nil {
		err := utils.ShouldErr(errors.New("argument 'ctx' is nil"))
		if err != nil {
			log.Error(err)
		}
	}
	if resolver.ctx == nil {
		resolver.ctx = ctx
	}
	return scoped.Context(resolver.ctx, scoped.Scope{
		Level: v1.SearchCategory_NAMESPACES,
		ID:    resolver.data.GetMetadata().GetId(),
	})
}

func (resolver *namespaceResolver) ImageVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "ImageVulnerabilities")
	return resolver.root.ImageVulnerabilities(resolver.namespaceScopeContext(ctx), args)
}

func (resolver *namespaceResolver) ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "ImageVulnerabilityCount")
	return resolver.root.ImageVulnerabilityCount(resolver.namespaceScopeContext(ctx), args)
}

func (resolver *namespaceResolver) ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "ImageVulnerabilityCounter")
	return resolver.root.ImageVulnerabilityCounter(resolver.namespaceScopeContext(ctx), args)
}

func (resolver *namespaceResolver) Secrets(ctx context.Context, args PaginatedQuery) ([]*secretResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Secrets")
	if err := readSecrets(ctx); err != nil {
		return nil, err
	}
	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getClusterNamespaceRawQuery())

	return resolver.root.Secrets(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

func (resolver *namespaceResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Deployments")
	return resolver.root.Deployments(resolver.namespaceScopeContext(ctx), args)
}

func (resolver *namespaceResolver) Cluster(ctx context.Context) (*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Cluster")
	return resolver.root.Cluster(resolver.namespaceScopeContext(ctx), struct{ graphql.ID }{graphql.ID(resolver.data.GetMetadata().GetClusterId())})
}

func (resolver *namespaceResolver) SecretCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "SecretCount")
	if err := readSecrets(ctx); err != nil {
		return 0, err
	}

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getClusterNamespaceRawQuery())

	return resolver.root.SecretCount(ctx, RawQuery{Query: &query})
}

func (resolver *namespaceResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "DeploymentCount")
	return resolver.root.DeploymentCount(resolver.namespaceScopeContext(ctx), args)
}

func (resolver *namespaceResolver) NetworkPolicyCount(ctx context.Context, _ RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "NetworkPolicyCount")
	if err := readNetPolicies(ctx); err != nil {
		return 0, err
	}

	networkPolicyCount, err := resolver.root.NetworkPoliciesStore.CountMatchingNetworkPolicies(
		ctx,
		resolver.data.GetMetadata().GetClusterId(),
		resolver.data.Metadata.GetName(),
	)
	if err != nil {
		return 0, errors.Wrap(err, "counting network policies")
	}

	return int32(networkPolicyCount), nil
}

func (resolver *namespaceResolver) Risk(ctx context.Context) (*riskResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Risk")
	if err := readDeploymentExtensions(ctx); err != nil {
		return nil, err
	}
	return resolver.root.wrapRisk(resolver.getNamespaceRisk(ctx))
}

func (resolver *namespaceResolver) getNamespaceRisk(ctx context.Context) (*storage.Risk, bool, error) {
	ns := resolver.data

	riskQuery := search.NewQueryBuilder().
		AddExactMatches(search.Namespace, ns.GetMetadata().GetName()).
		AddExactMatches(search.ClusterID, ns.GetMetadata().GetClusterId()).
		AddExactMatches(search.RiskSubjectType, storage.RiskSubjectType_DEPLOYMENT.String()).
		ProtoQuery()

	risks, err := resolver.root.RiskDataStore.SearchRawRisks(ctx, riskQuery)
	if err != nil {
		return nil, false, err
	}

	risks = filterDeploymentRisksOnScope(ctx, risks...)
	scrubRiskFactors(risks...)
	aggregateRiskScore := getAggregateRiskScore(risks...)
	if aggregateRiskScore == float32(0.0) {
		return nil, false, nil
	}

	risk := &storage.Risk{
		Score: aggregateRiskScore,
		Subject: &storage.RiskSubject{
			Id:        ns.GetMetadata().GetId(),
			Namespace: ns.GetMetadata().GetName(),
			ClusterId: ns.GetMetadata().GetClusterId(),
			Type:      storage.RiskSubjectType_NAMESPACE,
		},
	}

	id, err := riskDS.GetID(risk.GetSubject().GetId(), risk.GetSubject().GetType())
	if err != nil {
		return nil, false, err
	}
	risk.Id = id

	return risk, true, nil
}

func (resolver *namespaceResolver) LatestViolation(ctx context.Context, args RawQuery) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "LatestViolation")

	q, err := resolver.getNamespaceConjunctionQuery(args)
	if err != nil {
		return nil, nil
	}

	return getLatestViolationTime(ctx, resolver.root, q)
}

// PlottedImageVulnerabilities returns the data required by top risky entity scatter-plot on vuln mgmt dashboard
func (resolver *namespaceResolver) PlottedImageVulnerabilities(ctx context.Context, args RawQuery) (*PlottedImageVulnerabilitiesResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "PlottedImageVulnerabilities")
	return resolver.root.PlottedImageVulnerabilities(resolver.namespaceScopeContext(ctx), args)
}

func (resolver *namespaceResolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}
