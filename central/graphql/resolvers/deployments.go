package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/processindicator/service"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddExtraResolver("Deployment", `cluster: Cluster`),
		schema.AddExtraResolver("Deployment", `groupedProcesses: [ProcessNameGroup!]!`),
		schema.AddExtraResolver("Deployment", `alerts: [Alert!]!`),
		schema.AddQuery("deployment(id: ID): Deployment"),
		schema.AddQuery("deployments(query: String): [Deployment!]!"),
	)
}

// Deployment returns a GraphQL resolver for a given id
func (resolver *Resolver) Deployment(ctx context.Context, args struct{ *graphql.ID }) (*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapDeployment(resolver.DeploymentDataStore.GetDeployment(string(*args.ID)))
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
			resolver.DeploymentDataStore.ListDeployments())
	}
	return resolver.wrapListDeployments(
		resolver.DeploymentDataStore.SearchListDeployments(q))
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
	indicators, err := resolver.root.ProcessIndicatorStore.SearchRawProcessIndicators(query)
	return resolver.root.wrapProcessNameGroups(service.IndicatorsToGroupedResponses(indicators), err)
}

func (resolver *deploymentResolver) Alerts(ctx context.Context) ([]*alertResolver, error) {
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddStrings(search.DeploymentID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapAlerts(
		resolver.root.ViolationsDataStore.SearchRawAlerts(query))
}

func (resolver *Resolver) getDeployment(id string) *storage.Deployment {
	deployment, ok, err := resolver.DeploymentDataStore.GetDeployment(id)
	if err != nil || !ok {
		return nil
	}
	return deployment
}
