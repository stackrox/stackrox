package resolvers

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	const resolverName = "Pod"
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("pod(id: ID): Pod"),
		schema.AddQuery("pods(query: String, pagination: Pagination): [Pod!]!"),
		schema.AddQuery("podCount(query: String): Int!"),
		schema.AddExtraResolver(resolverName, "activeContainerInstancesCount: Int!"),
		schema.AddExtraResolver(resolverName, "policyViolationEvents: [PolicyViolationEvent!]!"),
		schema.AddExtraResolver(resolverName, "processActivityEvents: [ProcessActivityEvent!]!"),
		schema.AddExtraResolver(resolverName, "containerRestartEvents: [ContainerRestartEvent!]!"),
		schema.AddExtraResolver(resolverName, "containerTerminationEvents: [ContainerTerminationEvent!]!"),
		schema.AddExtraResolver(resolverName, "events: [DeploymentEvent!]!"),
	)
}

// Pod returns a GraphQL resolver for a given id.
func (resolver *Resolver) Pod(ctx context.Context, args struct{ *graphql.ID }) (*podResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Pod")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapPod(resolver.PodDataStore.GetPod(ctx, string(*args.ID)))
}

// Pods returns GraphQL resolvers for all pods.
func (resolver *Resolver) Pods(ctx context.Context, args PaginatedQuery) ([]*podResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Pods")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return resolver.wrapPods(resolver.PodDataStore.SearchRawPods(ctx, q))
}

// PodCount returns count of all pods across deployments
func (resolver *Resolver) PodCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "PodCount")
	if err := readDeployments(ctx); err != nil {
		return 0, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	results, err := resolver.PodDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// ActiveContainerInstancesCount returns the number of container instances that are currently active.
func (resolver *podResolver) ActiveContainerInstancesCount() int32 {
	return int32(len(resolver.data.LiveInstances))
}

// PolicyViolationEvents returns all policy violations associated with this pod.
func (resolver *podResolver) PolicyViolationEvents(ctx context.Context) ([]*PolicyViolationEventResolver, error) {
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}
	return nil, nil
}

// ProcessActivityEvents returns all the process activities associated with this pod.
func (resolver *podResolver) ProcessActivityEvents(ctx context.Context) ([]*ProcessActivityEventResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Pods, "ProcessActivityEvents")

	if err := readIndicators(ctx); err != nil {
		return nil, err
	}

	query := search.NewQueryBuilder().AddStrings(search.PodUID, resolver.data.GetId()).ProtoQuery()
	indicators, err := resolver.root.ProcessIndicatorStore.SearchRawProcessIndicators(ctx, query)
	if err != nil {
		return nil, err
	}

	processEvents := make([]*ProcessActivityEventResolver, 0, len(indicators))
	for _, indicator := range indicators {
		timestamp, err := types.TimestampFromProto(indicator.GetSignal().GetTime())
		if err != nil {
			log.Errorf("Unable to convert timestamp for indicator %s", indicator.GetSignal().GetName())
			continue
		}
		processEvents = append(processEvents, &ProcessActivityEventResolver{
			id:        indicator.GetId(),
			name:      indicator.GetSignal().GetName(),
			timestamp: timestamp,
			uid:       int32(indicator.GetSignal().GetUid()),
			parentUID: -1, // TODO: Get the parent UID
		})
	}
	return processEvents, nil
}

// ContainerRestartEvents returns all the container restart events associated with this pod.
func (resolver *podResolver) ContainerRestartEvents() []*ContainerRestartEventResolver {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Pods, "ContainerRestartEvents")

	var events []*ContainerRestartEventResolver
	liveInstances := resolver.data.GetLiveInstances()
	liveInstancesByName := make(map[string]*storage.ContainerInstance, len(liveInstances))
	for _, liveInstance := range liveInstances {
		liveInstancesByName[liveInstance.GetContainerName()] = liveInstance
	}

	for _, instances := range resolver.data.GetTerminatedInstances() {
		terminatedInstances := instances.GetInstances()
		if len(terminatedInstances) == 0 {
			continue
		}

		// The first terminated instance could not have been created from a restart.
		for i := 1; i < len(terminatedInstances); i++ {
			timestamp, err := types.TimestampFromProto(terminatedInstances[i].GetStarted())
			if err != nil {
				log.Errorf("Unable to convert timestamp for container instance %s", terminatedInstances[i].GetContainerName())
				continue
			}
			events = append(events, &ContainerRestartEventResolver{
				id:        terminatedInstances[i].GetInstanceId().GetId(),
				name:      terminatedInstances[i].GetContainerName(),
				timestamp: timestamp,
			})
		}

		// A current live instance can be due to a restart.
		containerName := terminatedInstances[0].GetContainerName()
		if instance, exists := liveInstancesByName[containerName]; exists {
			timestamp, err := types.TimestampFromProto(instance.GetStarted())
			if err != nil {
				log.Errorf("Unable to convert timestamp for container instance %s", instance.GetContainerName())
				continue
			}
			events = append(events, &ContainerRestartEventResolver{
				id:        instance.GetInstanceId().GetId(),
				name:      instance.GetContainerName(),
				timestamp: timestamp,
			})
			delete(liveInstancesByName, containerName)
		}
	}
	return events
}

// ContainerTerminationEvents returns all the container termination events associated with this pod.
func (resolver *podResolver) ContainerTerminationEvents() []*ContainerTerminationEventResolver {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Pods, "ContainerTerminationEvents")

	var events []*ContainerTerminationEventResolver
	for _, instances := range resolver.data.GetTerminatedInstances() {
		for _, instance := range instances.GetInstances() {
			timestamp, err := types.TimestampFromProto(instance.GetStarted())
			if err != nil {
				log.Errorf("Unable to convert timestamp for container instance %s", instance.GetContainerName())
				continue
			}
			events = append(events, &ContainerTerminationEventResolver{
				id:        instance.GetInstanceId().GetId(),
				name:      instance.GetContainerName(),
				timestamp: timestamp,
				exitCode:  instance.GetExitCode(),
				reason:    instance.GetTerminationReason(),
			})
		}
	}

	return events
}

// Events returns all events associated with this pod.
func (resolver *podResolver) Events(ctx context.Context) ([]*DeploymentEventResolver, error) {
	var events []*DeploymentEventResolver

	policyViolations, err := resolver.PolicyViolationEvents(ctx)
	if err != nil {
		return nil, err
	}
	for _, policyViolation := range policyViolations {
		events = append(events, &DeploymentEventResolver{policyViolation})
	}

	processActivities, err := resolver.ProcessActivityEvents(ctx)
	if err != nil {
		return nil, err
	}
	for _, processActivity := range processActivities {
		events = append(events, &DeploymentEventResolver{processActivity})
	}

	containerRestarts := resolver.ContainerRestartEvents()
	for _, containerRestart := range containerRestarts {
		events = append(events, &DeploymentEventResolver{containerRestart})
	}

	containerTerminations := resolver.ContainerTerminationEvents()
	for _, containerTermination := range containerTerminations {
		events = append(events, &DeploymentEventResolver{containerTermination})
	}

	return events, nil
}
