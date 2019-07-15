package resolvers

import (
	"context"
	"fmt"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/processindicator/service"
	"github.com/stackrox/rox/central/secret/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddExtraResolver("Deployment", `cluster: Cluster`),
		schema.AddExtraResolver("Deployment", `groupedProcesses: [ProcessNameGroup!]!`),
		schema.AddExtraResolver("Deployment", `alerts: [Alert!]!`),
		schema.AddExtraResolver("Deployment", `alertsCount: Int!`),
		schema.AddExtraResolver("Deployment", "complianceResults(query: String): [ControlResult!]!"),
		schema.AddExtraResolver("Deployment", "serviceAccountID: String!"),
		schema.AddExtraResolver("Deployment", `images: [Image!]!`),
		schema.AddExtraResolver("Deployment", `imagesCount: Int!`),
		schema.AddExtraResolver("Deployment", "secrets: [Secret!]!"),
		schema.AddExtraResolver("Deployment", "secretCount: Int!"),
		schema.AddQuery("deployment(id: ID): Deployment"),
		schema.AddQuery("deployments(query: String): [Deployment!]!"),
	)
}

// Deployment returns a GraphQL resolver for a given id
func (resolver *Resolver) Deployment(ctx context.Context, args struct{ *graphql.ID }) (*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapDeployment(resolver.DeploymentDataStore.GetDeployment(ctx, string(*args.ID)))
}

// Deployments returns GraphQL resolvers all deployments
func (resolver *Resolver) Deployments(ctx context.Context, args rawQuery) ([]*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	q, err := args.AsV1Query()
	if err != nil {
		return nil, err
	}
	if q == nil {
		return resolver.wrapListDeployments(
			resolver.DeploymentDataStore.ListDeployments(ctx))
	}
	return resolver.wrapListDeployments(
		resolver.DeploymentDataStore.SearchListDeployments(ctx, q))
}

// Cluster returns a GraphQL resolver for the cluster where this deployment runs
func (resolver *deploymentResolver) Cluster(ctx context.Context) (*clusterResolver, error) {
	clusterID := graphql.ID(resolver.data.GetClusterId())
	return resolver.root.Cluster(ctx, struct{ graphql.ID }{clusterID})
}

func (resolver *deploymentResolver) GroupedProcesses(ctx context.Context) ([]*processNameGroupResolver, error) {
	if err := readIndicators(ctx); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddStrings(search.DeploymentID, resolver.data.GetId()).ProtoQuery()
	indicators, err := resolver.root.ProcessIndicatorStore.SearchRawProcessIndicators(ctx, query)
	return resolver.root.wrapProcessNameGroups(service.IndicatorsToGroupedResponses(indicators), err)
}

func (resolver *deploymentResolver) Alerts(ctx context.Context) ([]*alertResolver, error) {
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddStrings(search.DeploymentID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapAlerts(
		resolver.root.ViolationsDataStore.SearchRawAlerts(ctx, query))
}

func (resolver *deploymentResolver) AlertsCount(ctx context.Context) (int32, error) {
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

// Secrets returns the total number of secrets for this deployment
func (resolver *deploymentResolver) Secrets(ctx context.Context) ([]*secretResolver, error) {
	secrets, err := resolver.getDeploymentSecrets(ctx)
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

// SecretCount returns the total number of secrets for this deployment
func (resolver *deploymentResolver) SecretCount(ctx context.Context) (int32, error) {
	secrets, err := resolver.getDeploymentSecrets(ctx)
	if err != nil {
		return 0, err
	}

	return int32(len(secrets)), nil
}

func (resolver *deploymentResolver) getDeploymentSecrets(ctx context.Context) ([]*secretResolver, error) {
	deployment := resolver.data

	// Find all the secret names referenced by the deployment
	secretsForDeploymentQuery := search.NewQueryBuilder().MarkHighlighted(search.SecretName).
		AddExactMatches(search.DeploymentID, resolver.data.GetId()).ProtoQuery()

	results, err := resolver.root.DeploymentDataStore.Search(ctx, secretsForDeploymentQuery)
	if len(results) == 0 || err != nil {
		return nil, err
	}

	field, exists := mappings.OptionsMap.Get(search.SecretName.String())
	if !exists {
		return nil, err
	}
	secrets := results[0].Matches[field.FieldPath]
	// For each secret name referenced by the deployment
	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, deployment.GetClusterId()).
		AddExactMatches(search.Namespace, deployment.GetNamespace()).
		AddStrings(search.SecretName, secrets...).
		AddStrings(search.SecretType, search.NegateQueryString(storage.SecretType_IMAGE_PULL_SECRET.String())).
		ProtoQuery()

	secretResults, err := resolver.root.SecretsDataStore.SearchRawSecrets(ctx, psr)
	if err != nil {
		return nil, err
	}

	return resolver.root.wrapSecrets(secretResults, nil)
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
	if err := readServiceAccounts(ctx); err != nil {
		return "", err
	}
	serviceAccounts, err := resolver.root.ServiceAccountsDataStore.ListServiceAccounts(ctx)
	if err != nil {
		return "", err
	}
	for _, serviceAccount := range serviceAccounts {
		if serviceAccount.ClusterId == resolver.ClusterId(ctx) && serviceAccount.Name == resolver.ServiceAccount(ctx) {
			return serviceAccount.Id, nil
		}
	}
	return "", errors.Wrap(nil, fmt.Sprintf("No matching service accounts found for deployment id: %s", resolver.Id(ctx)))
}

func (resolver *deploymentResolver) Images(ctx context.Context) ([]*imageResolver, error) {
	imageShas := resolver.getImageShas(ctx)
	return resolver.root.wrapImages(resolver.root.ImageDataStore.GetImagesBatch(ctx, imageShas))
}

func (resolver *deploymentResolver) ImagesCount(ctx context.Context) (int32, error) {
	imageShas := resolver.getImageShas(ctx)
	return int32(len(imageShas)), nil
}

func (resolver *deploymentResolver) getImageShas(ctx context.Context) []string {
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
