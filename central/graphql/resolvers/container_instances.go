package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("ContainerInstanceMock", []string{
			"id: ID!",
			"name: String!",
			"startTime: Int!",
		}),
		schema.AddQuery("containerInstances(id: ID): [ContainerInstanceMock!]!"),
		schema.AddExtraResolver("ContainerInstanceMock", "events: [DeploymentEvent!]!"),
	)
}

// ContainerInstanceMockResolver is a temporary dummy resolver for container instances.
type ContainerInstanceMockResolver struct {
	id        graphql.ID
	name      string
	startTime int32
}

// ContainerInstances returns GraphQL resolvers for all container instances associated with the given pod ID.
func (resolver *Resolver) ContainerInstances(ctx context.Context, _ struct{ *graphql.ID }) ([]*ContainerInstanceMockResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ContainerInstances")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	return []*ContainerInstanceMockResolver{
		{
			id:        "432143",
			name:      "scanner",
			startTime: 123123,
		},
		{
			id:        "23748732",
			name:      "scanner-db",
			startTime: 1234321,
		},
		{
			id:        "23748735",
			name:      "nginx",
			startTime: 1234325,
		},
	}, nil
}

// ID returns the container instance's ID.
func (resolver *ContainerInstanceMockResolver) ID(_ context.Context) graphql.ID {
	return resolver.id
}

// Name returns the container instance's name.
func (resolver *ContainerInstanceMockResolver) Name(_ context.Context) string {
	return resolver.name
}

// StartTime returns the container instance's start time.
func (resolver *ContainerInstanceMockResolver) StartTime(_ context.Context) int32 {
	return resolver.startTime
}

// Events returns the events associated with this container instance.
func (resolver *ContainerInstanceMockResolver) Events(_ context.Context) []*DeploymentEventResolver {
	return []*DeploymentEventResolver{
		{
			&ProcessActivityEventResolver{
				id:        "23432",
				name:      "/bin/bash",
				timestamp: 12343428,
				uid:       4000,
			},
		},
	}
}
