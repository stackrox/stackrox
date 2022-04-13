package resolvers

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/generated/storage"
	pkgMetrics "github.com/stackrox/stackrox/pkg/metrics"
	podUtils "github.com/stackrox/stackrox/pkg/pods/utils"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/utils"
)

func init() {
	schema := getBuilder()
	const groupResolverName = "ContainerNameGroup"
	utils.Must(
		schema.AddType(groupResolverName, []string{
			"id: ID!",
			"name: String!",
			"podId: String!",
			"startTime: Time",
			"containerInstances: [ContainerInstance!]!",
			"events: [DeploymentEvent!]!",
		}),
		schema.AddQuery("groupedContainerInstances(query: String): [ContainerNameGroup!]!"),
	)
}

// ContainerNameGroupResolver represents container instances grouped by their respective container name.
type ContainerNameGroupResolver struct {
	root                *Resolver
	name                string
	podID               podUtils.PodID
	deploymentID        string
	startTime           time.Time
	liveInstance        *containerInstanceResolver
	terminatedInstances []*containerInstanceResolver
	events              []*DeploymentEventResolver
}

// ID returns the group's ID.
func (resolver *ContainerNameGroupResolver) ID() graphql.ID {
	// The PodID + Container Name will give a unique ID.
	return graphql.ID(fmt.Sprintf("%s:%s", resolver.podID, resolver.name))
}

// Name returns the group's container name.
func (resolver *ContainerNameGroupResolver) Name() string {
	return resolver.name
}

// PodID returns the ID of the pod in which this group exists.
func (resolver *ContainerNameGroupResolver) PodID() string {
	return resolver.podID.String()
}

// StartTime returns the start time of the earliest container instance in the group.
func (resolver *ContainerNameGroupResolver) StartTime() *graphql.Time {
	return &graphql.Time{Time: resolver.startTime}
}

// Events returns all events associated with this container instance group.
func (resolver *ContainerNameGroupResolver) Events() []*DeploymentEventResolver {
	return resolver.events
}

// ContainerInstances returns the container instances in the group.
func (resolver *ContainerNameGroupResolver) ContainerInstances() []*containerInstanceResolver {
	instances := make([]*containerInstanceResolver, 0, len(resolver.terminatedInstances)+1)
	instances = append(instances, resolver.terminatedInstances...)
	instances = append(instances, resolver.liveInstance)
	return instances
}

// GroupedContainerInstances returns GraphQL resolvers for all container instances grouped by container name.
func (resolver *Resolver) GroupedContainerInstances(ctx context.Context, args RawQuery) ([]*ContainerNameGroupResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "GroupedContainerInstances")
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

	// Group each container instance group by the PodID and container name.
	groupByPodID := make(map[podUtils.PodID]map[string]*ContainerNameGroupResolver)
	for _, pod := range pods {
		podID := podUtils.GetPodIDFromStoragePod(pod)
		groupByName := make(map[string]*ContainerNameGroupResolver)
		groupByPodID[podID] = groupByName

		for _, instance := range pod.GetLiveInstances() {
			startTime, ok := convertTimestamp(instance.GetContainerName(), "container instance", instance.GetStarted())
			if !ok {
				continue
			}

			// Add a new group for each live instance.
			instanceResolver, err := resolver.wrapContainerInstance(instance, true, nil)
			if err != nil {
				log.Error(errors.Wrapf(err, "wrapping container instance %s", instance.GetContainerName()))
				continue
			}
			groupByName[instance.GetContainerName()] = &ContainerNameGroupResolver{
				root:         resolver,
				name:         instance.GetContainerName(),
				podID:        podID,
				deploymentID: pod.GetDeploymentId(),
				startTime:    startTime,
				liveInstance: instanceResolver,
			}
		}

		for _, terminatedInstancesList := range pod.GetTerminatedInstances() {
			instances := terminatedInstancesList.GetInstances()
			if len(instances) == 0 {
				continue
			}

			sort.SliceStable(instances, func(i, j int) bool {
				return instances[i].GetStarted().Compare(instances[j].GetStarted()) < 0
			})

			startTime, ok := convertTimestamp(instances[0].GetContainerName(), "container instance", instances[0].GetStarted())
			if !ok {
				continue
			}

			containerName := instances[0].GetContainerName()
			instancesResolver, err := resolver.wrapContainerInstances(instances, nil)
			if err != nil {
				log.Error(errors.Wrapf(err, "wrapping container instances %s", containerName))
				continue
			}
			if group, exists := groupByName[containerName]; exists {
				// Update the existing group.
				if startTime.Before(group.startTime) {
					group.startTime = startTime
				}
				group.terminatedInstances = instancesResolver
			} else {
				// Create a new group.
				groupByName[containerName] = &ContainerNameGroupResolver{
					root:                resolver,
					name:                containerName,
					podID:               podID,
					deploymentID:        pod.GetDeploymentId(),
					startTime:           startTime,
					terminatedInstances: instancesResolver,
				}
			}
		}
	}

	var groups []*ContainerNameGroupResolver
	for _, groupByName := range groupByPodID {
		for _, group := range groupByName {
			// This is the reason for making 'events' a normal field for the resolver instead of an extra resolver.
			// Kubernetes timestamps for containers only have second-precision, but our process events have at least
			// millisecond precision. Because of this, it is possible for the start time of a container to be after a
			// process's timestamp. We adjust the container's start time (as well as any restart and termination timestamps)
			// to alleviate the incorrectness due to the lack of precision.
			if err := populateEvents(ctx, group); err != nil {
				return nil, errors.Wrapf(err, "populating events for group %s", group.name)
			}
			if len(group.events) > 0 && group.startTime.After(group.events[0].Timestamp().Time) {
				group.startTime = group.events[0].Timestamp().Time
			}
			groups = append(groups, group)
		}
	}

	// Sort by container name.
	sort.SliceStable(groups, func(i, j int) bool {
		return groups[i].name < groups[j].name
	})

	return groups, nil
}

// policyViolationEvents returns all policy violations associated with this group of container instances.
func (resolver *ContainerNameGroupResolver) policyViolationEvents(ctx context.Context) ([]*PolicyViolationEventResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ContainerInstances, "PolicyViolationEvents")

	q := search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.DeploymentID, resolver.deploymentID).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.LifecycleStage, storage.LifecycleStage_RUNTIME.String()).ProtoQuery(),
	)

	predicateFn := func(alert *storage.Alert) bool {
		for _, proc := range alert.GetProcessViolation().GetProcesses() {
			// Filter by pod name because not all alerts may have PodUID (introduced in 42).
			if proc.GetPodId() == resolver.podID.Name && proc.GetContainerName() == resolver.name {
				return true
			}
		}
		return false
	}

	return resolver.root.getPolicyViolationEvents(ctx, q, predicateFn)
}

// processActivityEvents returns all the process activities associated with this group of container instances.
func (resolver *ContainerNameGroupResolver) processActivityEvents(ctx context.Context) ([]*ProcessActivityEventResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ContainerInstances, "ProcessActivityEvents")

	query := search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.DeploymentID, resolver.deploymentID).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.PodID, resolver.podID.Name).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.ContainerName, resolver.name).ProtoQuery(),
	)

	return resolver.root.getProcessActivityEvents(ctx, query)
}

// containerRestartEvents returns all the container restart events associated with this group of container instances.
func (resolver *ContainerNameGroupResolver) containerRestartEvents() []*ContainerRestartEventResolver {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ContainerInstances, "ContainerRestartEvents")

	events := make([]*ContainerRestartEventResolver, 0, len(resolver.terminatedInstances))
	for i := 1; i < len(resolver.terminatedInstances); i++ {
		started, ok := convertTimestamp(resolver.terminatedInstances[i].data.GetContainerName(), "container instance", resolver.terminatedInstances[i].data.GetStarted())
		if !ok {
			continue
		}
		events = append(events, &ContainerRestartEventResolver{
			id:        graphql.ID(resolver.terminatedInstances[i].data.GetInstanceId().GetId()),
			name:      resolver.name,
			timestamp: started,
		})
	}

	if resolver.liveInstance != nil && len(resolver.terminatedInstances) > 0 {
		started, ok := convertTimestamp(resolver.liveInstance.data.GetContainerName(), "container instance", resolver.liveInstance.data.GetStarted())
		if ok {
			events = append(events, &ContainerRestartEventResolver{
				id:        graphql.ID(resolver.liveInstance.data.GetInstanceId().GetId()),
				name:      resolver.name,
				timestamp: started,
			})
		}
	}

	return events
}

// containerTerminationEvents returns all the container termination events associated with this group of container instances.
func (resolver *ContainerNameGroupResolver) containerTerminationEvents() []*ContainerTerminationEventResolver {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ContainerInstances, "ContainerTerminationEvents")

	events := make([]*ContainerTerminationEventResolver, 0, len(resolver.terminatedInstances))
	for _, instance := range resolver.terminatedInstances {
		finished, ok := convertTimestamp(instance.data.GetContainerName(), "container instance", instance.data.GetFinished())
		if !ok {
			continue
		}
		events = append(events, &ContainerTerminationEventResolver{
			id:        graphql.ID(instance.data.GetInstanceId().GetId()),
			name:      resolver.name,
			timestamp: finished,
			exitCode:  instance.data.GetExitCode(),
			reason:    instance.data.GetTerminationReason(),
		})
	}

	return events
}

// populateEvents populates all of the events for the given container instance group sorted by timestamp.
func populateEvents(ctx context.Context, resolver *ContainerNameGroupResolver) error {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ContainerInstances, "Events")

	var events []*DeploymentEventResolver

	policyViolations, err := resolver.policyViolationEvents(ctx)
	if err != nil {
		return errors.Wrap(err, "retrieving policy violation events")
	}
	for _, policyViolation := range policyViolations {
		events = append(events, &DeploymentEventResolver{policyViolation})
	}

	processActivities, err := resolver.processActivityEvents(ctx)
	if err != nil {
		return errors.Wrap(err, "retrieving process activity events")
	}
	for _, processActivity := range processActivities {
		events = append(events, &DeploymentEventResolver{processActivity})
	}

	containerRestarts := resolver.containerRestartEvents()
	correctContainerRestartTimestamp(containerRestarts, processActivities)
	for _, containerRestart := range containerRestarts {
		events = append(events, &DeploymentEventResolver{containerRestart})
	}

	containerTerminations := resolver.containerTerminationEvents()
	correctContainerTerminationTimestamp(containerTerminations, processActivities)
	for _, containerTermination := range containerTerminations {
		events = append(events, &DeploymentEventResolver{containerTermination})
	}

	// Sort by timestamp.
	sort.SliceStable(events, func(i, j int) bool { return events[i].Timestamp().Before(events[j].Timestamp().Time) })

	resolver.events = events

	return nil
}
