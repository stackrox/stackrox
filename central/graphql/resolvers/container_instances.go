package resolvers

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	const resolverName = "ContainerInstance"
	utils.Must(
		schema.AddQuery("containerInstances(query: String): [ContainerInstance!]!"),
		schema.AddExtraResolver(resolverName, "events: [DeploymentEvent!]!"),
	)
}

// ContainerInstances returns GraphQL resolvers for all container instances.
func (resolver *Resolver) ContainerInstances(ctx context.Context, args RawQuery) ([]*containerInstanceResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ContainerInstances")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	pods, err := resolver.PodDataStore.SearchRawPods(ctx, query)
	if err != nil {
		return nil, err
	}
	var instances []*storage.ContainerInstance
	for _, pod := range pods {
		instances = append(instances, pod.GetLiveInstances()...)
		for _, terminatedInstancesList := range pod.GetTerminatedInstances() {
			instances = append(instances, terminatedInstancesList.GetInstances()...)
		}
	}

	return resolver.wrapContainerInstances(instances, nil)
}

// Events returns the events associated with this container instance.
func (resolver *containerInstanceResolver) Events(_ context.Context) []*DeploymentEventResolver {
	return []*DeploymentEventResolver{
		{
			&ProcessActivityEventResolver{
				id:   "23432",
				name: "/bin/bash",
				// 2 seconds from 'now'
				timestamp: time.Now().Add(2e9),
				uid:       4000,
			},
		},
	}
}
