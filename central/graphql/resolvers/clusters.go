package resolvers

import (
	"context"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/namespace"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("SubjectWithClusterID", []string{"clusterID: String!", "subject: Subject!"}),
		schema.AddType("PolicyStatus", []string{"status: String!", "failingPolicies: [Policy!]!"}),
		schema.AddQuery("clusters(query: String): [Cluster!]!"),
		schema.AddQuery("cluster(id: ID!): Cluster"),
		schema.AddExtraResolver("Cluster", `alerts: [Alert!]!`),
		schema.AddExtraResolver("Cluster", `alertCount: Int!`),
		schema.AddExtraResolver("Cluster", `deployments(query: String): [Deployment!]!`),
		schema.AddExtraResolver("Cluster", `deploymentCount: Int!`),
		schema.AddExtraResolver("Cluster", `nodes(query: String): [Node!]!`),
		schema.AddExtraResolver("Cluster", `nodeCount: Int!`),
		schema.AddExtraResolver("Cluster", `node(node: ID!): Node`),
		schema.AddExtraResolver("Cluster", `namespaces(query: String): [Namespace!]!`),
		schema.AddExtraResolver("Cluster", `namespace(name: String!): Namespace`),
		schema.AddExtraResolver("Cluster", `namespaceCount: Int!`),
		schema.AddExtraResolver("Cluster", "complianceResults(query: String): [ControlResult!]!"),
		schema.AddExtraResolver("Cluster", `k8sroles(query: String): [K8SRole!]!`),
		schema.AddExtraResolver("Cluster", `k8srole(role: ID!): K8SRole`),
		schema.AddExtraResolver("Cluster", `k8sroleCount: Int!`),
		schema.AddExtraResolver("Cluster", `serviceAccounts(query: String): [ServiceAccount!]!`),
		schema.AddExtraResolver("Cluster", `serviceAccount(sa: ID!): ServiceAccount`),
		schema.AddExtraResolver("Cluster", `serviceAccountCount: Int!`),
		schema.AddExtraResolver("Cluster", `subjects(query: String): [SubjectWithClusterID!]!`),
		schema.AddExtraResolver("Cluster", `subject(name: String!): SubjectWithClusterID!`),
		schema.AddExtraResolver("Cluster", `subjectCount: Int!`),
		schema.AddExtraResolver("Cluster", `images(query: String): [Image!]!`),
		schema.AddExtraResolver("Cluster", `imageCount: Int!`),
		schema.AddExtraResolver("Cluster", `policies(query: String): [Policy!]!`),
		schema.AddExtraResolver("Cluster", `policyCount: Int!`),
		schema.AddExtraResolver("Cluster", `policyStatus: PolicyStatus!`),
		schema.AddExtraResolver("Cluster", `secrets(query: String): [Secret!]!`),
		schema.AddExtraResolver("Cluster", `secretCount: Int!`),
		schema.AddExtraResolver("Cluster", `controlStatus: Boolean!`),
		schema.AddExtraResolver("Cluster", "controls(query: String): [ComplianceControl!]!"),
		schema.AddExtraResolver("Cluster", "failingControls(query: String): [ComplianceControl!]!"),
		schema.AddExtraResolver("Cluster", "passingControls(query: String): [ComplianceControl!]!"),
		schema.AddExtraResolver("Cluster", "complianceControlCount: ComplianceControlCount!"),
	)
}

// Cluster returns a GraphQL resolver for the given cluster
func (resolver *Resolver) Cluster(ctx context.Context, args struct{ graphql.ID }) (*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Cluster")
	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapCluster(resolver.ClusterDataStore.GetCluster(ctx, string(args.ID)))
}

// Clusters returns GraphQL resolvers for all clusters
func (resolver *Resolver) Clusters(ctx context.Context, args rawQuery) ([]*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Clusters")
	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1Query()
	if err != nil {
		return nil, err
	}
	if query == nil {
		return resolver.wrapClusters(resolver.ClusterDataStore.GetClusters(ctx))
	}
	return resolver.wrapClusters(resolver.ClusterDataStore.SearchRawClusters(ctx, query))
}

// Alerts returns GraphQL resolvers for all alerts on this cluster
func (resolver *clusterResolver) Alerts(ctx context.Context) ([]*alertResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Alerts")

	if err := readAlerts(ctx); err != nil {
		return nil, err // could return nil, nil to prevent errors from propagating.
	}
	query := resolver.getQuery()
	return resolver.root.wrapAlerts(
		resolver.root.ViolationsDataStore.SearchRawAlerts(ctx, query))
}

func (resolver *clusterResolver) AlertCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "AlertCount")
	if err := readAlerts(ctx); err != nil {
		return 0, err
	}
	query := resolver.getQuery()
	results, err := resolver.root.ViolationsDataStore.Search(ctx, query)
	if err != nil {
		return 0, err
	}

	return int32(len(results)), nil
}

// Deployments returns GraphQL resolvers for all deployments in this cluster
func (resolver *clusterResolver) Deployments(ctx context.Context, args rawQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Deployments")

	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	query, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapDeployments(resolver.root.DeploymentDataStore.SearchRawDeployments(ctx, query))
}

// DeploymentCount returns count of all deployments in this cluster
func (resolver *clusterResolver) DeploymentCount(ctx context.Context) (int32, error) {
	if err := readDeployments(ctx); err != nil {
		return 0, err
	}
	q := resolver.getQuery()
	results, err := resolver.root.DeploymentDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), err
}

// Nodes returns all nodes on the cluster
func (resolver *clusterResolver) Nodes(ctx context.Context, args rawQuery) ([]*nodeResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Nodes")

	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapNodes(resolver.root.NodeGlobalDataStore.SearchRawNodes(ctx, q))
}

// NodeCount returns count of all nodes on the cluster
func (resolver *clusterResolver) NodeCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "NodeCount")

	if err := readNodes(ctx); err != nil {
		return 0, err
	}
	store, err := resolver.root.NodeGlobalDataStore.GetClusterNodeStore(ctx, resolver.data.GetId(), false)
	if err != nil {
		return 0, err
	}

	nodeCount, err := store.CountNodes()
	if err != nil {
		return 0, err
	}

	return int32(nodeCount), nil
}

// Node returns a given node on a cluster
func (resolver *clusterResolver) Node(ctx context.Context, args struct{ Node graphql.ID }) (*nodeResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Node")

	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	store, err := resolver.root.NodeGlobalDataStore.GetClusterNodeStore(ctx, resolver.data.GetId(), false)
	if err != nil {
		return nil, err
	}
	node, err := store.GetNode(string(args.Node))
	return resolver.root.wrapNode(node, node != nil, err)
}

// Namespace returns a given namespace on a cluster.
func (resolver *clusterResolver) Namespaces(ctx context.Context, args rawQuery) ([]*namespaceResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Namespaces")

	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapNamespaces(namespace.ResolveByClusterID(ctx, resolver.data.GetId(),
		resolver.root.NamespaceDataStore, resolver.root.DeploymentDataStore, resolver.root.SecretsDataStore,
		resolver.root.NetworkPoliciesStore, q))
}

// Namespace returns a given namespace on a cluster.
func (resolver *clusterResolver) Namespace(ctx context.Context, args struct{ Name string }) (*namespaceResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Namespace")

	return resolver.root.NamespaceByClusterIDAndName(ctx, clusterIDAndNameQuery{
		ClusterID: graphql.ID(resolver.data.GetId()),
		Name:      args.Name,
	})
}

// NamespaceCount returns counts of namespaces on a cluster.
func (resolver *clusterResolver) NamespaceCount(ctx context.Context) (int32, error) {
	if err := readNamespaces(ctx); err != nil {
		return 0, err
	}
	q := resolver.getQuery()
	results, err := resolver.root.NamespaceDataStore.Search(ctx, q)
	if err != nil {
		return 0, nil
	}
	return int32(len(results)), nil
}

func (resolver *clusterResolver) ComplianceResults(ctx context.Context, args rawQuery) ([]*controlResultResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "ComplianceResults")

	if err := readCompliance(ctx); err != nil {
		return nil, err
	}

	runResults, err := resolver.root.ComplianceAggregator.GetResultsWithEvidence(ctx, args.String())
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	output.addClusterData(resolver.root, runResults, nil)
	output.addDeploymentData(resolver.root, runResults, nil)
	output.addNodeData(resolver.root, runResults, nil)
	return *output, nil
}

// K8sRoles returns GraphQL resolvers for all k8s roles
func (resolver *clusterResolver) K8sRoles(ctx context.Context, args rawQuery) ([]*k8SRoleResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "K8sRoles")

	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapK8SRoles(resolver.root.K8sRoleStore.SearchRawRoles(ctx, q))
}

// K8sRoleCount returns count of K8s roles in this cluster
func (resolver *clusterResolver) K8sRoleCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "K8sRoleCount")

	if err := readK8sRoles(ctx); err != nil {
		return 0, err
	}
	q := resolver.getQuery()
	results, err := resolver.root.K8sRoleStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// K8sRole returns clusterResolver GraphQL resolver for a given k8s role
func (resolver *clusterResolver) K8sRole(ctx context.Context, args struct{ Role graphql.ID }) (*k8SRoleResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "K8sRole")

	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).
		AddExactMatches(search.RoleID, string(args.Role)).ProtoQuery()

	roles, err := resolver.root.K8sRoleStore.SearchRawRoles(ctx, q)

	if err != nil {
		return nil, err
	}

	if len(roles) == 0 {
		return nil, nil
	}

	return resolver.root.wrapK8SRole(roles[0], true, nil)
}

// ServiceAccounts returns GraphQL resolvers for all service accounts in this cluster
func (resolver *clusterResolver) ServiceAccounts(ctx context.Context, args rawQuery) ([]*serviceAccountResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "ServiceAccounts")

	if err := readServiceAccounts(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapServiceAccounts(resolver.root.ServiceAccountsDataStore.SearchRawServiceAccounts(ctx, q))
}

// ServiceAccountCount returns count of Service Accounts in this cluster
func (resolver *clusterResolver) ServiceAccountCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "ServiceAccountCount")

	if err := readServiceAccounts(ctx); err != nil {
		return 0, err
	}
	q := resolver.getQuery()
	results, err := resolver.root.ServiceAccountsDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// ServiceAccount returns clusterResolver GraphQL resolver for a given service account
func (resolver *clusterResolver) ServiceAccount(ctx context.Context, args struct{ Sa graphql.ID }) (*serviceAccountResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "ServiceAccount")

	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).
		AddExactMatches(search.RoleID, string(args.Sa)).ProtoQuery()

	serviceAccounts, err := resolver.root.ServiceAccountsDataStore.SearchRawServiceAccounts(ctx, q)

	if err != nil {
		return nil, err
	}

	if len(serviceAccounts) == 0 {
		return nil, nil
	}

	return resolver.root.wrapServiceAccount(serviceAccounts[0], true, nil)
}

// Subjects returns GraphQL resolvers for all subjects in this cluster
func (resolver *clusterResolver) Subjects(ctx context.Context, args rawQuery) ([]*subjectWithClusterIDResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Subjects")

	subjectResolvers, err := resolver.root.wrapSubjects(resolver.getSubjects(ctx, args))
	if err != nil {
		return nil, err
	}
	return wrapSubjects(resolver.data.GetId(), resolver.data.GetName(), subjectResolvers), nil
}

// SubjectCount returns count of Users and Groups in this cluster
func (resolver *clusterResolver) SubjectCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "SubjectCount")

	subjects, err := resolver.getSubjects(ctx, rawQuery{})
	if err != nil {
		return 0, err
	}
	return int32(len(subjects)), nil
}

// ServiceAccount returns clusterResolver GraphQL resolver for a given service account
func (resolver *clusterResolver) Subject(ctx context.Context, args struct{ Name string }) (*subjectWithClusterIDResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Subject")

	bindings, err := resolver.getRoleBindings(ctx, rawQuery{})
	if err != nil {
		return nil, err
	}
	subject, err := resolver.root.wrapSubject(k8srbac.GetSubject(args.Name, bindings))
	if err != nil {
		return nil, err
	}
	return wrapSubject(resolver.data.GetId(), resolver.data.GetName(), subject), nil
}

func (resolver *clusterResolver) Images(ctx context.Context, args rawQuery) ([]*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Images")

	if err := readImages(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapImages(resolver.root.ImageDataStore.SearchRawImages(ctx, q))
}

func (resolver *clusterResolver) ImageCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "ImageCount")

	if err := readImages(ctx); err != nil {
		return 0, err
	}
	q := resolver.getQuery()
	results, err := resolver.root.ImageDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

func (resolver *clusterResolver) Policies(ctx context.Context, args rawQuery) ([]*policyResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Policies")

	if err := readPolicies(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	policies, err := resolver.root.PolicyDataStore.SearchRawPolicies(ctx, query)
	if err != nil {
		return nil, err
	}
	var filteredPolicies []*storage.Policy
	for _, policy := range policies {
		if resolver.policyAppliesToCluster(ctx, policy) {
			filteredPolicies = append(filteredPolicies, policy)
		}
	}
	return resolver.root.wrapPolicies(filteredPolicies, nil)
}

func (resolver *clusterResolver) policyAppliesToCluster(ctx context.Context, policy *storage.Policy) bool {
	// Global Policy
	if len(policy.Scope) == 0 {
		return true
	}
	// Clustered or namespaced scope policy
	for _, scope := range policy.Scope {
		if scope.GetCluster() != "" {
			if scope.GetCluster() == resolver.data.GetId() {
				return true
			}
		} else if scope.GetNamespace() != "" {
			q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).
				AddExactMatches(search.Namespace, scope.GetNamespace()).ProtoQuery()
			result, err := resolver.root.NamespaceDataStore.Search(ctx, q)
			if err != nil {
				continue
			}
			if len(result) != 0 {
				return true
			}
		} else {
			return true
		}
	}
	return false
}

func (resolver *clusterResolver) PolicyCount(ctx context.Context) (int32, error) {
	resolvers, err := resolver.Policies(ctx, rawQuery{})
	if err != nil {
		return 0, err
	}
	return int32(len(resolvers)), nil
}

// PolicyStatus returns true if there is no policy violation for this cluster
func (resolver *clusterResolver) PolicyStatus(ctx context.Context) (*policyStatusResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "PolicyStatus")

	alerts, err := resolver.getActiveDeployAlerts(ctx)
	if err != nil {
		return nil, err
	}

	if len(alerts) == 0 {
		return &policyStatusResolver{"pass", nil}, nil
	}

	policyIDs := set.NewStringSet()
	for _, alert := range alerts {
		policyIDs.Add(alert.GetPolicy().GetId())
	}

	policies, err := resolver.root.wrapPolicies(
		resolver.root.PolicyDataStore.SearchRawPolicies(ctx, search.NewQueryBuilder().AddDocIDs(policyIDs.AsSlice()...).ProtoQuery()))

	if err != nil {
		return nil, err
	}

	return &policyStatusResolver{"fail", policies}, nil
}

func (resolver *clusterResolver) Secrets(ctx context.Context, args rawQuery) ([]*secretResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Secrets")

	query, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapSecrets(resolver.root.SecretsDataStore.SearchRawSecrets(ctx, query))
}

func (resolver *clusterResolver) SecretCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "SecretCount")

	query := resolver.getQuery()
	result, err := resolver.root.SecretsDataStore.Search(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(len(result)), nil
}

func (resolver *clusterResolver) ControlStatus(ctx context.Context) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "ControlStatus")

	if err := readCompliance(ctx); err != nil {
		return false, err
	}
	r, err := resolver.getLastSuccessfulComplianceRunResult(ctx, []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER}, rawQuery{})
	if err != nil || r == nil {
		return false, err
	}
	if len(r) != 1 {
		return false, errors.Errorf("unexpected number of results: expected: 1, actual: %d", len(r))
	}
	return r[0].GetNumFailing() == 0, nil
}

func (resolver *clusterResolver) Controls(ctx context.Context, args rawQuery) ([]*complianceControlResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Controls")

	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	rs, err := resolver.getLastSuccessfulComplianceRunResult(ctx, []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER, v1.ComplianceAggregation_CONTROL}, args)
	if err != nil || rs == nil {
		return nil, err
	}
	resolvers, err := resolver.root.wrapComplianceControls(getComplianceControlsFromAggregationResults(rs, any, resolver.root.ComplianceStandardStore))
	if err != nil {
		return nil, err
	}
	return resolvers, nil
}

func (resolver *clusterResolver) PassingControls(ctx context.Context, args rawQuery) ([]*complianceControlResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "PassingControls")

	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	rs, err := resolver.getLastSuccessfulComplianceRunResult(ctx, []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER, v1.ComplianceAggregation_CONTROL}, args)
	if err != nil || rs == nil {
		return nil, err
	}
	resolvers, err := resolver.root.wrapComplianceControls(getComplianceControlsFromAggregationResults(rs, passing, resolver.root.ComplianceStandardStore))
	if err != nil {
		return nil, err
	}
	return resolvers, nil
}

func (resolver *clusterResolver) FailingControls(ctx context.Context, args rawQuery) ([]*complianceControlResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "FailingControls")

	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	rs, err := resolver.getLastSuccessfulComplianceRunResult(ctx, []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER, v1.ComplianceAggregation_CONTROL}, args)
	if err != nil || rs == nil {
		return nil, err
	}
	resolvers, err := resolver.root.wrapComplianceControls(getComplianceControlsFromAggregationResults(rs, failing, resolver.root.ComplianceStandardStore))
	if err != nil {
		return nil, err
	}
	return resolvers, nil
}

func (resolver *clusterResolver) ComplianceControlCount(ctx context.Context) (*complianceControlCountResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "ComplianceControlCount")
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	r, err := resolver.getLastSuccessfulComplianceRunResult(ctx, []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER}, rawQuery{})
	if err != nil {
		return nil, err
	}
	if r == nil {
		return &complianceControlCountResolver{}, nil
	}
	if len(r) != 1 {
		return &complianceControlCountResolver{}, errors.Errorf("unexpected number of results: expected: 1, actual: %d", len(r))
	}
	return &complianceControlCountResolver{failingCount: r[0].GetNumFailing(), passingCount: r[0].GetNumPassing()}, nil
}

func (resolver *clusterResolver) getLastSuccessfulComplianceRunResult(ctx context.Context, scope []v1.ComplianceAggregation_Scope, args rawQuery) ([]*v1.ComplianceAggregation_Result, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	standardIDs, err := getStandardIDs(ctx, resolver.root.ComplianceStandardStore)
	if err != nil {
		return nil, err
	}
	hasComplianceSuccessfullyRun, err := resolver.root.ComplianceDataStore.IsComplianceRunSuccessfulOnCluster(ctx, resolver.data.GetId(), standardIDs)
	if err != nil || !hasComplianceSuccessfullyRun {
		return nil, err
	}
	query, err := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).RawQuery()
	if err != nil {
		return nil, err
	}
	if args.Query != nil {
		query = strings.Join([]string{query, *(args.Query)}, "+")
	}
	r, _, _, err := resolver.root.ComplianceAggregator.Aggregate(ctx, query, scope, v1.ComplianceAggregation_CONTROL)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (resolver *clusterResolver) getActiveDeployAlerts(ctx context.Context) ([]*storage.ListAlert, error) {
	cluster := resolver.data

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, cluster.GetId()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddStrings(search.LifecycleStage, storage.LifecycleStage_DEPLOY.String()).ProtoQuery()

	return resolver.root.ViolationsDataStore.SearchListAlerts(ctx, q)
}
