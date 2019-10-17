package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processindicator/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("deployment(id: ID): Deployment"),
		schema.AddQuery("deployments(query: String, pagination: Pagination): [Deployment!]!"),
		schema.AddQuery("deploymentCount(query: String): Int!"),
		schema.AddExtraResolver("Deployment", `cluster: Cluster`),
		schema.AddExtraResolver("Deployment", `namespaceObject: Namespace`),
		schema.AddExtraResolver("Deployment", `serviceAccountObject: ServiceAccount`),
		schema.AddExtraResolver("Deployment", `groupedProcesses: [ProcessNameGroup!]!`),
		schema.AddExtraResolver("Deployment", `deployAlerts: [Alert!]!`),
		schema.AddExtraResolver("Deployment", `deployAlertCount: Int!`),
		schema.AddExtraResolver("Deployment", `failingPolicies(query: String): [Policy!]!`),
		schema.AddExtraResolver("Deployment", `failingPolicyCount(query: String): Int!`),
		schema.AddExtraResolver("Deployment", "complianceResults(query: String): [ControlResult!]!"),
		schema.AddExtraResolver("Deployment", "serviceAccountID: String!"),
		schema.AddExtraResolver("Deployment", `images(query: String): [Image!]!`),
		schema.AddExtraResolver("Deployment", `imageCount: Int!`),
		schema.AddExtraResolver("Deployment", `imageComponents: [EmbeddedImageScanComponent!]!`),
		schema.AddExtraResolver("Deployment", `imageComponentCount: Int!`),
		schema.AddExtraResolver("Deployment", `vulns: [EmbeddedVulnerability!]!`),
		schema.AddExtraResolver("Deployment", `vulnCount: Int!`),
		schema.AddExtraResolver("Deployment", "secrets(query: String): [Secret!]!"),
		schema.AddExtraResolver("Deployment", "secretCount: Int!"),
		schema.AddExtraResolver("Deployment", "policyStatus(query: String) : String!"),
	)
}

// Deployment returns a GraphQL resolver for a given id
func (resolver *Resolver) Deployment(ctx context.Context, args struct{ *graphql.ID }) (*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Deployment")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapDeployment(resolver.DeploymentDataStore.GetDeployment(ctx, string(*args.ID)))
}

// Deployments returns GraphQL resolvers all deployments
func (resolver *Resolver) Deployments(ctx context.Context, args paginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Deployments")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return resolver.wrapDeployments(
		resolver.DeploymentDataStore.SearchRawDeployments(ctx, q))
}

// DeploymentCount returns count all deployments across infrastructure
func (resolver *Resolver) DeploymentCount(ctx context.Context, args rawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "DeploymentCount")
	if err := readDeployments(ctx); err != nil {
		return 0, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	results, err := resolver.DeploymentDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// Cluster returns a GraphQL resolver for the cluster where this deployment runs
func (resolver *deploymentResolver) Cluster(ctx context.Context) (*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Deployments, "Cluster")
	if err := readClusters(ctx); err != nil {
		return nil, err
	}

	clusterID := graphql.ID(resolver.data.GetClusterId())
	return resolver.root.Cluster(ctx, struct{ graphql.ID }{clusterID})
}

// NamespaceObject returns a GraphQL resolver for the namespace where this deployment runs
func (resolver *deploymentResolver) NamespaceObject(ctx context.Context) (*namespaceResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Deployments, "NamespaceObject")

	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	namespaceID := graphql.ID(resolver.data.GetNamespaceId())
	return resolver.root.Namespace(ctx, struct{ graphql.ID }{namespaceID})
}

// ServiceAccountObject returns a GraphQL resolver for the service account associated with this deployment
func (resolver *deploymentResolver) ServiceAccountObject(ctx context.Context) (*serviceAccountResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Deployments, "ServiceAccountObject")

	if err := readServiceAccounts(ctx); err != nil {
		return nil, err
	}
	serviceAccountName := resolver.data.GetServiceAccount()
	results, err := resolver.root.ServiceAccountsDataStore.SearchRawServiceAccounts(ctx, search.NewQueryBuilder().AddExactMatches(
		search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetNamespace()).
		AddExactMatches(search.ServiceAccountName, serviceAccountName).ProtoQuery())

	if err != nil || results == nil {
		return nil, err
	}

	return resolver.root.wrapServiceAccount(results[0], true, err)
}

func (resolver *deploymentResolver) GroupedProcesses(ctx context.Context) ([]*processNameGroupResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Deployments, "GroupedProcesses")

	if err := readIndicators(ctx); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddStrings(search.DeploymentID, resolver.data.GetId()).ProtoQuery()
	indicators, err := resolver.root.ProcessIndicatorStore.SearchRawProcessIndicators(ctx, query)
	return resolver.root.wrapProcessNameGroups(service.IndicatorsToGroupedResponses(indicators), err)
}

func (resolver *deploymentResolver) DeployAlerts(ctx context.Context) ([]*alertResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Deployments, "DeployAlerts")

	if err := readAlerts(ctx); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddStrings(search.DeploymentID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapAlerts(
		resolver.root.ViolationsDataStore.SearchRawAlerts(ctx, query))
}

func (resolver *deploymentResolver) DeployAlertCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Deployments, "DeployAlertsCount")

	if err := readAlerts(ctx); err != nil {
		return 0, err // could return nil, nil to prevent errors from propagating.
	}
	query := search.NewQueryBuilder().AddStrings(search.DeploymentID, resolver.data.GetId()).ProtoQuery()
	results, err := resolver.root.ViolationsDataStore.Search(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// FailingPolicies returns policy resolvers for policies failing on this deployment
func (resolver *deploymentResolver) FailingPolicies(ctx context.Context, args rawQuery) ([]*policyResolver, error) {
	if err := readPolicies(ctx); err != nil {
		return nil, err
	}
	query, err := resolver.getFailingAlertsQuery(args)
	if err != nil {
		return nil, err
	}
	alerts, err := resolver.root.ViolationsDataStore.SearchRawAlerts(ctx, query)
	if err != nil {
		return nil, err
	}
	var policies []*storage.Policy
	set := set.NewStringSet()
	for _, alert := range alerts {
		if set.Add(alert.GetPolicy().GetId()) {
			policies = append(policies, alert.GetPolicy())
		}
	}
	return resolver.root.wrapPolicies(policies, nil)
}

// FailingPolicyCount returns count of policies failing on this deployment
func (resolver *deploymentResolver) FailingPolicyCount(ctx context.Context, args rawQuery) (int32, error) {
	if err := readPolicies(ctx); err != nil {
		return 0, err
	}
	query, err := resolver.getFailingAlertsQuery(args)
	if err != nil {
		return 0, err
	}
	alerts, err := resolver.root.ViolationsDataStore.SearchListAlerts(ctx, query)
	if err != nil {
		return 0, nil
	}
	set := set.NewStringSet()
	for _, alert := range alerts {
		set.Add(alert.GetPolicy().GetId())
	}
	return int32(set.Cardinality()), nil
}

// Secrets returns the total number of secrets for this deployment
func (resolver *deploymentResolver) Secrets(ctx context.Context, args rawQuery) ([]*secretResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Deployments, "Secrets")

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	secrets, err := resolver.getDeploymentSecrets(ctx, q)
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

// SecretCount returns the total number of secrets for this deployment
func (resolver *deploymentResolver) SecretCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Deployments, "SecretCount")

	secrets, err := resolver.getDeploymentSecrets(ctx, search.EmptyQuery())
	if err != nil {
		return 0, err
	}

	return int32(len(secrets)), nil
}

func (resolver *deploymentResolver) getDeploymentSecrets(ctx context.Context, q *v1.Query) ([]*secretResolver, error) {
	if err := readSecrets(ctx); err != nil {
		return nil, err
	}
	deployment := resolver.data
	secretSet := set.NewStringSet()
	for _, container := range deployment.GetContainers() {
		for _, secret := range container.GetSecrets() {
			secretSet.Add(secret.GetName())
		}
	}
	if secretSet.Cardinality() == 0 {
		return []*secretResolver{}, nil
	}
	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, deployment.GetClusterId()).
		AddExactMatches(search.Namespace, deployment.GetNamespace()).
		AddStrings(search.SecretName, secretSet.AsSlice()...).
		ProtoQuery()
	secrets, err := resolver.root.SecretsDataStore.SearchRawSecrets(ctx, psr)
	if err != nil {
		return nil, err
	}
	for _, secret := range secrets {
		resolver.root.getDeploymentRelationships(ctx, secret)
	}
	return resolver.root.wrapSecrets(secrets, nil)
}

func (resolver *Resolver) getDeployment(ctx context.Context, id string) *storage.Deployment {
	deployment, ok, err := resolver.DeploymentDataStore.GetDeployment(ctx, id)
	if err != nil || !ok {
		return nil
	}
	return deployment
}

func (resolver *deploymentResolver) ComplianceResults(ctx context.Context, args rawQuery) ([]*controlResultResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}

	runResults, err := resolver.root.ComplianceAggregator.GetResultsWithEvidence(ctx, args.String())
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	deploymentID := resolver.data.GetId()
	output.addDeploymentData(resolver.root, runResults, func(d *storage.Deployment, _ *v1.ComplianceControl) bool {
		return d.GetId() == deploymentID
	})

	return *output, nil
}

func (resolver *deploymentResolver) ServiceAccountID(ctx context.Context) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Deployments, "ServiceAccountID")

	if err := readServiceAccounts(ctx); err != nil {
		return "", err
	}

	clusterID := resolver.ClusterId(ctx)
	serviceAccountName := resolver.ServiceAccount(ctx)

	q := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).
		AddExactMatches(search.ServiceAccountName, serviceAccountName).
		ProtoQuery()

	results, err := resolver.root.ServiceAccountsDataStore.Search(ctx, q)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", errors.Wrap(nil, fmt.Sprintf("No matching service accounts found for deployment id: %s", resolver.Id(ctx)))
	}
	return results[0].ID, nil
}

func (resolver *deploymentResolver) Images(ctx context.Context, args rawQuery) ([]*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Deployments, "Images")

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	imageShas := resolver.getImageShas(ctx)
	imageShaQuery := search.NewQueryBuilder().AddDocIDs(imageShas...).ProtoQuery()

	return resolver.root.wrapImages(resolver.root.ImageDataStore.SearchRawImages(ctx,
		search.NewConjunctionQuery(imageShaQuery, q)))
}

func (resolver *deploymentResolver) ImageCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Deployments, "ImageCount")

	imageShas := resolver.getImageShas(ctx)
	return int32(len(imageShas)), nil
}

func (resolver *deploymentResolver) ImageComponents(ctx context.Context) ([]*EmbeddedImageScanComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Vulns")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	imageShas := resolver.getImageShas(ctx)
	imageShaQuery := search.NewQueryBuilder().AddDocIDs(imageShas...).ProtoQuery()
	images, err := resolver.root.ImageDataStore.SearchRawImages(ctx, imageShaQuery)
	if err != nil {
		return nil, err
	}
	return mapImagesToComponentResolvers(resolver.root, images, search.EmptyQuery())
}

func (resolver *deploymentResolver) ImageComponentCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "VulnCount")
	if err := readImages(ctx); err != nil {
		return 0, err
	}

	imageShas := resolver.getImageShas(ctx)
	imageShaQuery := search.NewQueryBuilder().AddDocIDs(imageShas...).ProtoQuery()
	images, err := resolver.root.ImageDataStore.SearchRawImages(ctx, imageShaQuery)
	if err != nil {
		return 0, err
	}

	vulns, err := mapImagesToComponentResolvers(resolver.root, images, search.EmptyQuery())
	if err != nil {
		return 0, err
	}
	return int32(len(vulns)), nil
}

func (resolver *deploymentResolver) Vulns(ctx context.Context) ([]*EmbeddedVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "Vulns")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	imageShas := resolver.getImageShas(ctx)
	imageShaQuery := search.NewQueryBuilder().AddDocIDs(imageShas...).ProtoQuery()
	images, err := resolver.root.ImageDataStore.SearchRawImages(ctx, imageShaQuery)
	if err != nil {
		return nil, err
	}
	return mapImagesToVulnerabilityResolvers(resolver.root, images, search.EmptyQuery())
}

func (resolver *deploymentResolver) VulnCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "VulnCount")
	if err := readImages(ctx); err != nil {
		return 0, err
	}

	imageShas := resolver.getImageShas(ctx)
	imageShaQuery := search.NewQueryBuilder().AddDocIDs(imageShas...).ProtoQuery()
	images, err := resolver.root.ImageDataStore.SearchRawImages(ctx, imageShaQuery)
	if err != nil {
		return 0, err
	}

	vulns, err := mapImagesToVulnerabilityResolvers(resolver.root, images, search.EmptyQuery())
	if err != nil {
		return 0, err
	}
	return int32(len(vulns)), nil
}

func (resolver *deploymentResolver) PolicyStatus(ctx context.Context, args rawQuery) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Deployments, "PolicyStatus")

	alertExists, err := resolver.unresolvedAlertsExists(ctx, args)
	if err != nil {
		return "", err
	}
	if alertExists {
		return "fail", nil
	}
	return "pass", nil
}

func (resolver *deploymentResolver) getImageShas(ctx context.Context) []string {
	if err := readImages(ctx); err != nil {
		return nil
	}

	imageShas := set.NewStringSet()

	deployment := resolver.data
	containers := deployment.GetContainers()
	for _, c := range containers {
		if c.GetImage().GetId() != "" {
			imageShas.Add(c.GetImage().GetId())
		}
	}
	return imageShas.AsSlice()
}

func (resolver *deploymentResolver) unresolvedAlertsExists(ctx context.Context, args rawQuery) (bool, error) {
	if err := readAlerts(ctx); err != nil {
		return false, err
	}
	q, err := resolver.getFailingAlertsQuery(args)
	if err != nil {
		return false, err
	}
	q.Pagination = &v1.QueryPagination{Limit: 1}
	results, err := resolver.root.ViolationsDataStore.Search(ctx, q)
	if err != nil {
		return false, err
	}
	return len(results) > 0, nil
}

func (resolver *deploymentResolver) getQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.DeploymentID, resolver.data.GetId()).ProtoQuery()
}

func (resolver *deploymentResolver) getConjunctionQuery(args rawQuery) (*v1.Query, error) {
	q1 := resolver.getQuery()
	if args.String() == "" {
		return q1, nil
	}
	q2, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return search.NewConjunctionQuery(q2, q1), nil
}

func (resolver *deploymentResolver) getFailingAlertsQuery(args rawQuery) (*v1.Query, error) {
	q, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	return search.NewConjunctionQuery(q, search.NewQueryBuilder().AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()), nil
}
