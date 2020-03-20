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
			"containerName: String!",
			"startTime: Time",
		}),
		schema.AddQuery("containerInstances(id: ID): [ContainerInstanceMock!]!"),
		schema.AddExtraResolver("ContainerInstanceMock", "events: [DeploymentEvent!]!"),
	)
}

// ContainerInstanceMockResolver is a temporary dummy resolver for container instances.
type ContainerInstanceMockResolver struct {
	id            graphql.ID
	containerName string
	startTime     time.Time
}

// ContainerInstances returns GraphQL resolvers for all container instances associated with the given pod ID.
func (resolver *Resolver) ContainerInstances(ctx context.Context, _ struct{ *graphql.ID }) ([]*ContainerInstanceMockResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ContainerInstances")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	now := time.Now()
	return []*ContainerInstanceMockResolver{
		{
			id:            "432143",
			containerName: "scanner",
			startTime:     now,
		},
		{
			id:            "23748732",
			containerName: "scanner-db",
			// 30 milliseconds after 'now'
			startTime: now.Add(3e7),
		},
		{
			id:            "23748735",
			containerName: "nginx",
			// 1 second after 'now'
			startTime: now.Add(1e9),
		},
	}, nil
}

// ID returns the container instance's ID.
func (resolver *ContainerInstanceMockResolver) ID(_ context.Context) graphql.ID {
	return resolver.id
}

// ContainerName returns the container instance's name.
func (resolver *ContainerInstanceMockResolver) ContainerName(_ context.Context) string {
	return resolver.containerName
}

// StartTime returns the container instance's start time.
func (resolver *ContainerInstanceMockResolver) StartTime(_ context.Context) *graphql.Time {
	return &graphql.Time{Time: resolver.startTime}
}

// Events returns the events associated with this container instance.
func (resolver *ContainerInstanceMockResolver) Events(_ context.Context) []*DeploymentEventResolver {
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
