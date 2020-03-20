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
		schema.AddType("PodMock", []string{
			"id: ID!",
			"name: String!",
			"startTime: Time",
			"inactive: Boolean!",
			"containerCount: Int!",
		}),
		schema.AddQuery("pod(id: ID): PodMock"),
		schema.AddQuery("pods(query: String, pagination: Pagination): [PodMock!]!"),
		schema.AddExtraResolver("PodMock", "events: [DeploymentEvent!]!"),
	)
}

// PodMockResolver is a temporary dummy resolver for pods.
type PodMockResolver struct {
	id             graphql.ID
	name           string
	startTime      time.Time
	inactive       bool
	containerCount int32
}

// Pod returns a GraphQL resolver for a given id.
func (resolver *Resolver) Pod(ctx context.Context, args struct{ *graphql.ID }) (*PodMockResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Pod")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	return &PodMockResolver{
		id:             *args.ID,
		name:           "nginx-7db9fccd9b-92hfs",
		startTime:      time.Now(),
		inactive:       false,
		containerCount: 3,
	}, nil
}

// Pods returns GraphQL resolvers for all pods.
func (resolver *Resolver) Pods(ctx context.Context, _ PaginatedQuery) ([]*PodMockResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Pods")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	now := time.Now()
	return []*PodMockResolver{
		{
			id:             "0",
			name:           "scanner-84c6678b58-2dj5j",
			startTime:      now,
			inactive:       false,
			containerCount: 3,
		},
		{
			id:   "1",
			name: "scanner-db-6dcf8894d7-k2mcw",
			// 1 millisecond from 'now'
			startTime:      now.Add(1e6),
			inactive:       true,
			containerCount: 3,
		},
		{
			id:   "2",
			name: "nginx-7db9fccd9b-92hfs",
			// 3 millisecond from 'now'
			startTime:      now.Add(3e6),
			inactive:       false,
			containerCount: 3,
		},
		{
			id:   "3",
			name: "nginx-7db9fccd9b-sl2mk",
			// 10 milliseconds from 'now'
			startTime:      now.Add(1e7),
			inactive:       false,
			containerCount: 3,
		},
		{
			id:   "4",
			name: "nginx-7db9fccd9b-xkqv9",
			// 1 second from 'now'
			startTime:      now.Add(1e9),
			inactive:       false,
			containerCount: 3,
		},
		{
			id:   "5",
			name: "nginx-7db9fccd9b-9w8bz",
			// 3 seconds from 'now'
			startTime:      now.Add(3e9),
			inactive:       true,
			containerCount: 3,
		},
		{
			id:   "6",
			name: "scanner-db-6dcf8894d7-k2mcw",
			// 5 minutes from 'now'
			startTime:      now.Add(3e11),
			inactive:       true,
			containerCount: 3,
		},
		{
			id:   "7",
			name: "scanner-db-6dcf8894d7-k2mcw",
			// 20 microseconds from 'now'
			startTime:      now.Add(2e4),
			inactive:       true,
			containerCount: 3,
		},
		{
			id:   "8",
			name: "scanner-db-6dcf8894d7-k2mcw",
			// 17 milliseconds from 'now'
			startTime:      now.Add(1.7e7),
			inactive:       true,
			containerCount: 3,
		},
		{
			id:   "9",
			name: "scanner-db-6dcf8894d7-k2mcw",
			// 1 hour from 'now'
			startTime:      now.Add(3.6e12),
			inactive:       true,
			containerCount: 3,
		},
	}, nil
}

// ID returns the pod's ID.
func (resolver *PodMockResolver) ID(_ context.Context) graphql.ID {
	return resolver.id
}

// Name returns the pod's name.
func (resolver *PodMockResolver) Name(_ context.Context) string {
	return resolver.name
}

// StartTime returns the pod's start time.
func (resolver *PodMockResolver) StartTime(_ context.Context) *graphql.Time {
	return &graphql.Time{Time: resolver.startTime}
}

// Inactive says whether or not this pod is inactive.
func (resolver *PodMockResolver) Inactive(_ context.Context) bool {
	return resolver.inactive
}

// ContainerCount gets the number of containers in this pod's history.
func (resolver *PodMockResolver) ContainerCount(_ context.Context) int32 {
	return resolver.containerCount
}

// Events returns the events associated with this pod.
func (resolver *PodMockResolver) Events(_ context.Context) []*DeploymentEventResolver {
	now := time.Now()
	return []*DeploymentEventResolver{
		{
			&ContainerTerminationEventResolver{
				id:   "1234",
				name: "nginx",
				// 1 minute from 'now'
				timestamp: now.Add(6e10),
				exitCode:  0,
				reason:    "Completed",
			},
		},
		{
			&ContainerTerminationEventResolver{
				id:   "1235",
				name: "nginx",
				// 1 second from 'now'
				timestamp: now.Add(1e9),
				exitCode:  137,
				reason:    "Finished",
			},
		},
		{
			&ContainerRestartEventResolver{
				id:   "1236",
				name: "nginx",
				// 20 milliseconds from 'now'
				timestamp: now.Add(2e7),
			},
		},
		{
			&ProcessActivityEventResolver{
				id:   "23432",
				name: "/bin/bash",
				// 500 milliseconds from 'now'
				timestamp: now.Add(5e8),
				uid:       4000,
			},
		},
	}
}
