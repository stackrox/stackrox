package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/schema"
	"github.com/stackrox/rox/central/processindicator/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

func init() {
	schema.AddResolver(&v1.Deployment{}, `cluster: Cluster`)
	schema.AddResolver(&v1.Deployment{}, `groupedProcesses: [ProcessNameGroup!]!`)
	schema.AddResolver(&v1.Deployment{}, `alerts: [Alert!]!`)
	schema.AddQuery("deployment(id: ID): Deployment")
	schema.AddQuery("deployments(): [Deployment!]!")
}

// Deployment returns a GraphQL resolver for a given id
func (resolver *Resolver) Deployment(ctx context.Context, args struct{ *graphql.ID }) (*deploymentResolver, error) {
	if err := deploymentAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapDeployment(resolver.DeploymentDataStore.GetDeployment(string(*args.ID)))
}

// Deployments returns GraphQL resolvers all deployments
func (resolver *Resolver) Deployments(ctx context.Context) ([]*deploymentResolver, error) {
	if err := deploymentAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapDeployments(resolver.DeploymentDataStore.GetDeployments())
}

// Cluster returns a GraphQL resolver for the cluster where this deployment runs
func (resolver *deploymentResolver) Cluster(ctx context.Context) (*clusterResolver, error) {
	clusterID := graphql.ID(resolver.data.GetClusterId())
	return resolver.root.Cluster(ctx, struct{ graphql.ID }{clusterID})
}

func (resolver *deploymentResolver) GroupedProcesses(ctx context.Context) ([]*processNameGroupResolver, error) {
	if err := indicatorAuth(ctx); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddStrings(search.DeploymentID, resolver.data.GetId()).ProtoQuery()
	indicators, err := resolver.root.ProcessIndicatorStore.SearchRawProcessIndicators(query)
	return resolver.root.wrapProcessNameGroups(service.IndicatorsToGroupedResponses(indicators), err)
}

func (resolver *deploymentResolver) Alerts(ctx context.Context) ([]*alertResolver, error) {
	if err := alertAuth(ctx); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddStrings(search.DeploymentID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapAlerts(
		resolver.root.ViolationsDataStore.SearchRawAlerts(query))
}
