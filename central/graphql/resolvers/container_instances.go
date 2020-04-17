package resolvers

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	podUtils "github.com/stackrox/rox/pkg/pods/utils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
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
		}),
		schema.AddQuery("groupedContainerInstances(query: String): [ContainerNameGroup!]!"),
		schema.AddExtraResolver(groupResolverName, "policyViolationEvents: [PolicyViolationEvent!]!"),
		schema.AddExtraResolver(groupResolverName, "processActivityEvents: [ProcessActivityEvent!]!"),
		schema.AddExtraResolver(groupResolverName, "containerRestartEvents: [ContainerRestartEvent!]!"),
		schema.AddExtraResolver(groupResolverName, "containerTerminationEvents: [ContainerTerminationEvent!]!"),
		schema.AddExtraResolver(groupResolverName, "events: [DeploymentEvent!]!"),
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
			groups = append(groups, group)
		}
	}

	// Sort by container name.
	sort.SliceStable(groups, func(i, j int) bool {
		return groups[i].name < groups[j].name
	})

	return groups, nil
}

// PolicyViolationEvents returns all policy violations associated with this group of container instances.
func (resolver *ContainerNameGroupResolver) PolicyViolationEvents(ctx context.Context) ([]*PolicyViolationEventResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ContainerInstances, "PolicyViolationEvents")

	// We search by PodID (name) to filter out processes involving other pods.
	// PodID is guaranteed to be unique within a deployment during the pod's
	// lifetime and all process indicators should have this field.
	// Not all process indicators will have PodUID, so we cannot filter based on that.
	// Also use ContainerName to ensure we only get results for the relevant resolver.
	q := search.NewConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.DeploymentID, resolver.deploymentID).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.LifecycleStage, storage.LifecycleStage_RUNTIME.String()).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.PodID, resolver.podID.Name).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.ContainerName, resolver.name).ProtoQuery(),
	)

	return resolver.root.getPolicyViolationEvents(ctx, q)
}

// ProcessActivityEvents returns all the process activities associated with this group of container instances.
func (resolver *ContainerNameGroupResolver) ProcessActivityEvents(ctx context.Context) ([]*ProcessActivityEventResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ContainerInstances, "ProcessActivityEvents")

	query := search.NewConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.DeploymentID, resolver.deploymentID).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.PodID, resolver.podID.Name).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.ContainerName, resolver.name).ProtoQuery(),
	)

	return resolver.root.getProcessActivityEvents(ctx, query)
}

// ContainerRestartEvents returns all the container restart events associated with this group of container instances.
func (resolver *ContainerNameGroupResolver) ContainerRestartEvents() []*ContainerRestartEventResolver {
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

// ContainerTerminationEvents returns all the container termination events associated with this group of container instances.
func (resolver *ContainerNameGroupResolver) ContainerTerminationEvents() []*ContainerTerminationEventResolver {
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

// Events returns all events associated with this pod.
func (resolver *ContainerNameGroupResolver) Events(ctx context.Context) ([]*DeploymentEventResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ContainerInstances, "Events")

	var events []*DeploymentEventResolver

	policyViolations, err := resolver.PolicyViolationEvents(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving policy violation events")
	}
	for _, policyViolation := range policyViolations {
		events = append(events, &DeploymentEventResolver{policyViolation})
	}

	processActivities, err := resolver.ProcessActivityEvents(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving process activity events")
	}
	for _, processActivity := range processActivities {
		events = append(events, &DeploymentEventResolver{processActivity})
	}

	for _, containerRestart := range resolver.ContainerRestartEvents() {
		events = append(events, &DeploymentEventResolver{containerRestart})
	}

	for _, containerTermination := range resolver.ContainerTerminationEvents() {
		events = append(events, &DeploymentEventResolver{containerTermination})
	}

	return events, nil
}
