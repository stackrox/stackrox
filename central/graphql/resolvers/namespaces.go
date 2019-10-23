package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/namespace"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/scopecomp"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("namespaces(query: String): [Namespace!]!"),
		schema.AddQuery("namespace(id: ID!): Namespace"),
		schema.AddQuery("namespaceByClusterIDAndName(clusterID: ID!, name: String!): Namespace"),
		schema.AddExtraResolver("Namespace", "complianceResults(query: String): [ControlResult!]!"),
		schema.AddExtraResolver("Namespace", `subjects(query: String): [Subject!]!`),
		schema.AddExtraResolver("Namespace", `subjectCount: Int!`),
		schema.AddExtraResolver("Namespace", `serviceAccountCount: Int!`),
		schema.AddExtraResolver("Namespace", `serviceAccounts(query: String): [ServiceAccount!]!`),
		schema.AddExtraResolver("Namespace", `k8sroleCount: Int!`),
		schema.AddExtraResolver("Namespace", `k8sroles(query: String): [K8SRole!]!`),
		schema.AddExtraResolver("Namespace", `policyCount(query: String): Int!`),
		schema.AddExtraResolver("Namespace", `policyStatus: PolicyStatus!`),
		schema.AddExtraResolver("Namespace", `policies(query: String): [Policy!]!`),
		schema.AddExtraResolver("Namespace", `images(query: String): [Image!]!`),
		schema.AddExtraResolver("Namespace", `imageCount: Int!`),
		schema.AddExtraResolver("Namespace", `imageComponents: [EmbeddedImageScanComponent!]!`),
		schema.AddExtraResolver("Namespace", `imageComponentCount: Int!`),
		schema.AddExtraResolver("Namespace", `vulns: [EmbeddedVulnerability!]!`),
		schema.AddExtraResolver("Namespace", `vulnCount: Int!`),
		schema.AddExtraResolver("Namespace", `vulnCounter: VulnerabilityCounter!`),
		schema.AddExtraResolver("Namespace", `secrets(query: String): [Secret!]!`),
		schema.AddExtraResolver("Namespace", `deployments(query: String): [Deployment!]!`),
		schema.AddExtraResolver("Namespace", "cluster: Cluster!"),
		schema.AddExtraResolver("Namespace", `secretCount: Int!`),
		schema.AddExtraResolver("Namespace", `deploymentCount: Int!`),
		schema.AddExtraResolver("Namespace", `risk: Risk`),
	)
}

// Namespace returns a GraphQL resolver for the given namespace.
func (resolver *Resolver) Namespace(ctx context.Context, args struct{ graphql.ID }) (*namespaceResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Namespace")
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapNamespace(namespace.ResolveByID(ctx, string(args.ID), resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
}

// Namespaces returns GraphQL resolvers for all namespaces based on an optional query.
func (resolver *Resolver) Namespaces(ctx context.Context, args rawQuery) ([]*namespaceResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Namespaces")
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return resolver.wrapNamespaces(namespace.ResolveByQuery(ctx, query, resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
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
	return resolver.wrapNamespace(namespace.ResolveByClusterIDAndName(ctx, string(args.ClusterID), args.Name, resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
}

func (resolver *namespaceResolver) ComplianceResults(ctx context.Context, args rawQuery) ([]*controlResultResolver, error) {
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
	output.addDeploymentData(resolver.root, runResults, func(d *storage.Deployment, _ *v1.ComplianceControl) bool {
		return d.GetNamespaceId() == nsID
	})

	return *output, nil
}

// SubjectCount returns the count of Subjects which have any permission on this namespace or the cluster it belongs to
func (resolver *namespaceResolver) SubjectCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "SubjectCount")
	if err := readK8sSubjects(ctx); err != nil {
		return 0, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return 0, err
	}
	subjects, err := resolver.getSubjects(ctx, search.EmptyQuery())
	if err != nil {
		return 0, err
	}
	return int32(len(subjects)), nil
}

// Subjects returns the Subjects which have any permission in namespace or cluster wide
func (resolver *namespaceResolver) Subjects(ctx context.Context, args rawQuery) ([]*subjectResolver, error) {
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
	subjects, err := resolver.getSubjects(ctx, baseQuery)
	if err != nil {
		return nil, err
	}
	for _, subject := range subjects {
		resolvers = append(resolvers, &subjectResolver{resolver.root, subject})
	}
	return resolvers, nil
}

// ServiceAccountCount returns the count of ServiceAccounts which have any permission on this cluster namespace
func (resolver *namespaceResolver) ServiceAccountCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "ServiceAccountCount")
	if err := readServiceAccounts(ctx); err != nil {
		return 0, err
	}
	q := resolver.getClusterNamespaceQuery()
	results, err := resolver.root.ServiceAccountsDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// ServiceAccounts returns the ServiceAccounts which have any permission on this cluster namespace
func (resolver *namespaceResolver) ServiceAccounts(ctx context.Context, args rawQuery) ([]*serviceAccountResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "ServiceAccounts")
	if err := readServiceAccounts(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapServiceAccounts(resolver.root.ServiceAccountsDataStore.SearchRawServiceAccounts(ctx, q))
}

// K8sRoleCount returns count of K8s roles in this cluster namespace
func (resolver *namespaceResolver) K8sRoleCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "K8sRoleCount")
	if err := readK8sRoles(ctx); err != nil {
		return 0, err
	}
	q := resolver.getClusterNamespaceQuery()
	results, err := resolver.root.K8sRoleStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// K8sRoles returns count of K8s roles in this cluster namespace
func (resolver *namespaceResolver) K8sRoles(ctx context.Context, args rawQuery) ([]*k8SRoleResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "K8sRoles")
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapK8SRoles(resolver.root.K8sRoleStore.SearchRawRoles(ctx, q))
}

func (resolver *namespaceResolver) ImageCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "ImageCount")
	if err := readNamespaces(ctx); err != nil {
		return 0, err
	}
	q := resolver.getClusterNamespaceQuery()
	results, err := resolver.root.ImageDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

func (resolver *namespaceResolver) filterPoliciesApplicableToNamespace(policies []*storage.Policy) []*storage.Policy {
	var filteredPolicies []*storage.Policy
	clusterID := resolver.data.GetMetadata().GetClusterId()
	for _, policy := range policies {
		if resolver.policyAppliesToNamespace(policy, clusterID) {
			filteredPolicies = append(filteredPolicies, policy)
		}
	}
	return filteredPolicies
}

func (resolver *namespaceResolver) getNamespacePolicies(ctx context.Context, args rawQuery) ([]*storage.Policy, error) {
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	policies, err := resolver.root.PolicyDataStore.SearchRawPolicies(ctx, q)
	if err != nil {
		return nil, err
	}
	return resolver.filterPoliciesApplicableToNamespace(policies), nil
}

// PolicyCount returns count of policies applicable to this namespace
func (resolver *namespaceResolver) PolicyCount(ctx context.Context, args rawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "PolicyCount")
	if err := readPolicies(ctx); err != nil {
		return 0, err
	}
	policies, err := resolver.getNamespacePolicies(ctx, args)
	if err != nil {
		return 0, err
	}
	return int32(len(policies)), nil
}

// Policies returns all the policies applicable to this namespace
func (resolver *namespaceResolver) Policies(ctx context.Context, args rawQuery) ([]*policyResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Policies")
	if err := readPolicies(ctx); err != nil {
		return nil, err
	}
	return resolver.root.wrapPolicies(resolver.getNamespacePolicies(ctx, args))
}

func (resolver *namespaceResolver) policyAppliesToNamespace(policy *storage.Policy, clusterID string) bool {
	// Global Policy
	if len(policy.GetScope()) == 0 {
		return true
	}
	// Clustered or namespace scope policy, evaluate all scopes
	for _, scope := range policy.GetScope() {
		cs, err := scopecomp.CompileScope(scope)
		if err != nil {
			utils.Should(errors.Wrap(err, "could not compile scope"))
			continue
		}
		if scope.GetCluster() != "" && cs.MatchesCluster(clusterID) && cs.MatchesNamespace(resolver.data.Metadata.GetName()) {
			return true
		}
		if cs.MatchesNamespace(resolver.data.Metadata.GetName()) {
			return true
		}
	}
	return false
}

// PolicyStatus returns true if there is no policy violation for this cluster
func (resolver *namespaceResolver) PolicyStatus(ctx context.Context) (*policyStatusResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "PolicyStatus")
	alerts, err := resolver.getActiveDeployAlerts(ctx)
	if err != nil {
		return nil, err
	}

	if len(alerts) == 0 {
		return &policyStatusResolver{resolver.root, "pass", nil}, nil
	}

	policyIDs := set.NewStringSet()
	for _, alert := range alerts {
		policyIDs.Add(alert.GetPolicy().GetId())
	}

	return &policyStatusResolver{resolver.root, "fail", policyIDs.AsSlice()}, nil
}

func (resolver *namespaceResolver) getActiveDeployAlerts(ctx context.Context) ([]*storage.ListAlert, error) {
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}

	namespace := resolver.data

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, namespace.GetMetadata().GetClusterId()).
		AddExactMatches(search.Namespace, namespace.GetMetadata().GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddStrings(search.LifecycleStage, storage.LifecycleStage_DEPLOY.String()).ProtoQuery()

	return resolver.root.ViolationsDataStore.SearchListAlerts(ctx, q)
}

func (resolver *namespaceResolver) Images(ctx context.Context, args rawQuery) ([]*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Images")
	if err := readImages(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapImages(imageLoader.FromQuery(ctx, q))
}

func (resolver *namespaceResolver) ImageComponents(ctx context.Context) ([]*EmbeddedImageScanComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Vulns")

	if err := readImages(ctx); err != nil {
		return nil, err
	}

	images, err := resolver.root.ImageDataStore.SearchRawImages(ctx, resolver.getClusterNamespaceQuery())
	if err != nil {
		return nil, err
	}
	return mapImagesToComponentResolvers(resolver.root, images, search.EmptyQuery())
}

func (resolver *namespaceResolver) ImageComponentCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "VulnCount")
	if err := readImages(ctx); err != nil {
		return 0, err
	}

	images, err := resolver.root.ImageDataStore.SearchRawImages(ctx, resolver.getClusterNamespaceQuery())
	if err != nil {
		return 0, err
	}

	vulns, err := mapImagesToComponentResolvers(resolver.root, images, search.EmptyQuery())
	if err != nil {
		return 0, err
	}
	return int32(len(vulns)), nil
}

func (resolver *namespaceResolver) Vulns(ctx context.Context) ([]*EmbeddedVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Vulns")

	if err := readImages(ctx); err != nil {
		return nil, err
	}

	images, err := resolver.root.ImageDataStore.SearchRawImages(ctx, resolver.getClusterNamespaceQuery())
	if err != nil {
		return nil, err
	}
	return mapImagesToVulnerabilityResolvers(resolver.root, images, search.EmptyQuery())
}

func (resolver *namespaceResolver) VulnCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "VulnCount")
	if err := readImages(ctx); err != nil {
		return 0, err
	}

	images, err := resolver.root.ImageDataStore.SearchRawImages(ctx, resolver.getClusterNamespaceQuery())
	if err != nil {
		return 0, err
	}

	vulns, err := mapImagesToVulnerabilityResolvers(resolver.root, images, search.EmptyQuery())
	if err != nil {
		return 0, err
	}
	return int32(len(vulns)), nil
}

func (resolver *namespaceResolver) VulnCounter(ctx context.Context) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "VulnCounter")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	images, err := resolver.root.ImageDataStore.SearchRawImages(ctx, resolver.getClusterNamespaceQuery())
	if err != nil {
		return nil, err
	}
	return mapImagesToVulnerabilityCounter(images), nil
}

func (resolver *namespaceResolver) Secrets(ctx context.Context, args rawQuery) ([]*secretResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Secrets")
	if err := readSecrets(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapSecrets(resolver.root.SecretsDataStore.SearchRawSecrets(ctx, q))
}

func (resolver *namespaceResolver) Deployments(ctx context.Context, args rawQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Deployments")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapDeployments(resolver.root.DeploymentDataStore.SearchRawDeployments(ctx, q))
}

func (resolver *namespaceResolver) Cluster(ctx context.Context) (*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Cluster")
	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	return resolver.root.wrapCluster(resolver.root.ClusterDataStore.GetCluster(ctx, resolver.data.GetMetadata().GetClusterId()))
}

func (resolver *namespaceResolver) SecretCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "SecretCount")
	if err := readSecrets(ctx); err != nil {
		return 0, err
	}
	return resolver.data.GetNumSecrets(), nil
}

func (resolver *namespaceResolver) DeploymentCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "DeploymentCount")
	if err := readDeployments(ctx); err != nil {
		return 0, err
	}
	return resolver.data.GetNumDeployments(), nil
}

func (resolver *namespaceResolver) getConjunctionQuery(args rawQuery) (*v1.Query, error) {
	q1 := resolver.getClusterNamespaceQuery()
	if args.String() == "" {
		return q1, nil
	}
	q2, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return search.NewConjunctionQuery(q2, q1), nil
}

func (resolver *namespaceResolver) getClusterNamespaceQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.Metadata.GetName()).ProtoQuery()
}

func (resolver *namespaceResolver) Risk(ctx context.Context) (*riskResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Namespaces, "Risk")
	if err := readRisks(ctx); err != nil {
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
