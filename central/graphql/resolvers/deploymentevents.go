package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/processbaseline"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/containerid"
	processBaselinePkg "github.com/stackrox/rox/pkg/processbaseline"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddInterfaceType("DeploymentEvent", []string{
			"id: ID!",
			"name: String!",
			"timestamp: Time",
		}),
		schema.AddType("ContainerTerminationEvent", []string{
			"id: ID!",
			"name: String!",
			"timestamp: Time",
			"exitCode: Int!",
			"reason: String!",
		}, "DeploymentEvent"),
		schema.AddType("ContainerRestartEvent", []string{
			"id: ID!",
			"name: String!",
			"timestamp: Time",
		}, "DeploymentEvent"),
		schema.AddType("ProcessActivityEvent", []string{
			"id: ID!",
			"name: String!",
			"timestamp: Time",
			"args: String!",
			"uid: Int!",
			"parentName: String",
			"parentUid: Int!",
			"inBaseline: Boolean!",
		}, "DeploymentEvent"),
		schema.AddType("PolicyViolationEvent", []string{
			"id: ID!",
			"name: String!",
			"timestamp: Time",
		}, "DeploymentEvent"),
	)
}

// DeploymentEvent is the parent interface for events.
type DeploymentEvent interface {
	ID() graphql.ID
	Name() string
	Timestamp() *graphql.Time
}

// DeploymentEventResolver is the parent resolver for event resolvers.
type DeploymentEventResolver struct {
	DeploymentEvent
}

// ToContainerTerminationEvent converts a deployment event to a container termination event.
func (resolver *DeploymentEventResolver) ToContainerTerminationEvent() (*ContainerTerminationEventResolver, bool) {
	e, ok := resolver.DeploymentEvent.(*ContainerTerminationEventResolver)
	return e, ok
}

// ToContainerRestartEvent converts a deployment event to a container restart event.
func (resolver *DeploymentEventResolver) ToContainerRestartEvent() (*ContainerRestartEventResolver, bool) {
	e, ok := resolver.DeploymentEvent.(*ContainerRestartEventResolver)
	return e, ok
}

// ToProcessActivityEvent converts a deployment event to a process event.
func (resolver *DeploymentEventResolver) ToProcessActivityEvent() (*ProcessActivityEventResolver, bool) {
	e, ok := resolver.DeploymentEvent.(*ProcessActivityEventResolver)
	return e, ok
}

// ToPolicyViolationEvent converts a deployment event to a policy violation event.
func (resolver *DeploymentEventResolver) ToPolicyViolationEvent() (*PolicyViolationEventResolver, bool) {
	e, ok := resolver.DeploymentEvent.(*PolicyViolationEventResolver)
	return e, ok
}

// ContainerTerminationEventResolver represents a container termination (failure or graceful) event.
type ContainerTerminationEventResolver struct {
	id        graphql.ID
	name      string
	timestamp time.Time
	exitCode  int32
	reason    string
}

// ID returns the event's ID.
func (resolver *ContainerTerminationEventResolver) ID() graphql.ID {
	return resolver.id
}

// Name returns the event's name.
func (resolver *ContainerTerminationEventResolver) Name() string {
	return resolver.name
}

// Timestamp returns the event's timestamp.
func (resolver *ContainerTerminationEventResolver) Timestamp() *graphql.Time {
	return &graphql.Time{Time: resolver.timestamp}
}

// ExitCode returns the failed container's exist code.
func (resolver *ContainerTerminationEventResolver) ExitCode() int32 {
	return resolver.exitCode
}

// Reason returns the reason for the container's failure.
func (resolver *ContainerTerminationEventResolver) Reason() string {
	return resolver.reason
}

// ContainerRestartEventResolver represents a container restart event.
type ContainerRestartEventResolver struct {
	id        graphql.ID
	name      string
	timestamp time.Time
}

// ID returns the event's ID.
func (resolver *ContainerRestartEventResolver) ID() graphql.ID {
	return resolver.id
}

// Name returns the event's name.
func (resolver *ContainerRestartEventResolver) Name() string {
	return resolver.name
}

// Timestamp returns the event's timestamp.
func (resolver *ContainerRestartEventResolver) Timestamp() *graphql.Time {
	return &graphql.Time{Time: resolver.timestamp}
}

// ProcessActivityEventResolver represents a process start event.
type ProcessActivityEventResolver struct {
	id                  graphql.ID
	name                string
	timestamp           time.Time
	args                string
	uid                 int32
	parentName          *string
	parentUID           int32
	containerInstanceID string
	canReadBaseline     bool
	inBaseline          bool
}

// ID returns the event's ID.
func (resolver *ProcessActivityEventResolver) ID() graphql.ID {
	return resolver.id
}

// Name returns the event's name.
func (resolver *ProcessActivityEventResolver) Name() string {
	return resolver.name
}

// Timestamp returns the event's timestamp.
func (resolver *ProcessActivityEventResolver) Timestamp() *graphql.Time {
	return &graphql.Time{Time: resolver.timestamp}
}

// Args returns the process's arguments.
func (resolver *ProcessActivityEventResolver) Args() string {
	return resolver.args
}

// UID returns the process's UID.
func (resolver *ProcessActivityEventResolver) UID() int32 {
	return resolver.uid
}

// ParentName returns the process's parent's name, if it exists, and null otherwise.
func (resolver *ProcessActivityEventResolver) ParentName() *string {
	return resolver.parentName
}

// ParentUID returns the process's parent's UID.
// Any value greater than or equal to 0 indicates a parent's UID.
// -1 indicates the parent's UID is unknown or the process does not have a parent.
// This should be used in tandem with ParentName if the exact state of the UID is desired.
//
// If ParentName is not null, a ParentUID of -1 implies the UID is unknown.
// If ParentName is null, then ParentUID will always be -1.
func (resolver *ProcessActivityEventResolver) ParentUID() int32 {
	return resolver.parentUID
}

// InBaseline returns true if this process is in baseline.
func (resolver *ProcessActivityEventResolver) InBaseline() bool {
	if resolver.canReadBaseline {
		return resolver.inBaseline
	}
	// Default to true if the requester cannot read the baseline.
	return true
}

func (resolver *Resolver) getProcessActivityEvents(ctx context.Context, query *v1.Query) ([]*ProcessActivityEventResolver, error) {
	if err := readDeploymentExtensions(ctx); err != nil {
		return nil, err
	}

	indicators, err := resolver.ProcessIndicatorStore.SearchRawProcessIndicators(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving process indicators from search")
	}

	processEvents := make([]*ProcessActivityEventResolver, 0, len(indicators))
	baselines := make(map[string]*set.StringSet)
	// This determines if we should read baseline information.
	// nil means we can.
	canReadBaseline := readDeploymentExtensions(ctx) == nil
	for _, indicator := range indicators {
		var keyStr, procName string
		if canReadBaseline {
			key := &storage.ProcessBaselineKey{
				ClusterId:     indicator.GetClusterId(),
				Namespace:     indicator.GetNamespace(),
				DeploymentId:  indicator.GetDeploymentId(),
				ContainerName: indicator.GetContainerName(),
			}
			keyStr = key.String()
			procName = processBaselinePkg.BaselineItemFromProcess(indicator)
			if procName != "" {
				if _, exists := baselines[keyStr]; !exists {
					baseline, exists, err := resolver.BaselineDataStore.GetProcessBaseline(ctx, key)
					if err != nil {
						log.Error(errors.Wrapf(err, "retrieving baseline data for process %s", indicator.GetSignal().GetName()))
						continue
					}
					if !exists {
						continue
					}

					baselines[keyStr] = processbaseline.Processes(baseline, processbaseline.RoxOrUserLocked)
				}
			}
		}

		timestamp, ok := convertTimestamp(indicator.GetSignal().GetName(), "indicator", indicator.GetSignal().GetTime())
		if !ok {
			continue
		}
		// -1 indicates we do not have parent UID information (either no parent exists or we do not know its UID).
		var parentName *string
		parentUID := int32(-1)
		if lineageInfo := indicator.GetSignal().GetLineageInfo(); len(lineageInfo) > 0 {
			// This process's direct parent should be the first entry.
			name := lineageInfo[0].GetParentExecFilePath()
			parentName = &name
			parentUID = int32(lineageInfo[0].GetParentUid())
		} else if lineage := indicator.GetSignal().GetLineage(); len(lineage) > 0 {
			// This process's direct parent should be the first entry.
			parentName = &lineage[0]
		}
		processEvents = append(processEvents, &ProcessActivityEventResolver{
			id:                  graphql.ID(indicator.GetId()),
			name:                indicator.GetSignal().GetExecFilePath(),
			timestamp:           timestamp,
			args:                indicator.GetSignal().GetArgs(),
			uid:                 int32(indicator.GetSignal().GetUid()),
			parentName:          parentName,
			parentUID:           parentUID,
			containerInstanceID: indicator.GetSignal().GetContainerId(),
			canReadBaseline:     canReadBaseline,
			inBaseline:          baselines[keyStr] == nil || baselines[keyStr].Contains(procName),
		})
	}
	return processEvents, nil
}

func correctContainerRestartTimestamp(restartResolvers []*ContainerRestartEventResolver,
	processResolvers []*ProcessActivityEventResolver) {
	instanceIDToTimestamp := make(map[string]time.Time, len(restartResolvers))
	for _, restartEvent := range restartResolvers {
		id := containerid.ShortContainerIDFromInstanceID(string(restartEvent.id))
		instanceIDToTimestamp[id] = restartEvent.timestamp
	}
	for _, processEvent := range processResolvers {
		earliestTimestamp, exists := instanceIDToTimestamp[processEvent.containerInstanceID]
		if exists && processEvent.timestamp.Before(earliestTimestamp) {
			instanceIDToTimestamp[processEvent.containerInstanceID] = processEvent.timestamp
		}
	}
	for _, restartEvent := range restartResolvers {
		id := containerid.ShortContainerIDFromInstanceID(string(restartEvent.id))
		restartEvent.timestamp = instanceIDToTimestamp[id]
	}
}

func correctContainerTerminationTimestamp(terminationResolvers []*ContainerTerminationEventResolver,
	processResolvers []*ProcessActivityEventResolver) {
	instanceIDToTimestamp := make(map[string]time.Time, len(terminationResolvers))
	for _, terminationEvent := range terminationResolvers {
		id := containerid.ShortContainerIDFromInstanceID(string(terminationEvent.id))
		instanceIDToTimestamp[id] = terminationEvent.timestamp
	}
	for _, processEvent := range processResolvers {
		latestTimestamp, exists := instanceIDToTimestamp[processEvent.containerInstanceID]
		if exists && processEvent.timestamp.After(latestTimestamp) {
			instanceIDToTimestamp[processEvent.containerInstanceID] = processEvent.timestamp
		}
	}
	for _, terminationEvent := range terminationResolvers {
		id := containerid.ShortContainerIDFromInstanceID(string(terminationEvent.id))
		terminationEvent.timestamp = instanceIDToTimestamp[id]
	}
}

// PolicyViolationEventResolver represents a policy violation event.
type PolicyViolationEventResolver struct {
	id        graphql.ID
	name      string
	timestamp time.Time
}

// ID returns the event's ID.
func (resolver *PolicyViolationEventResolver) ID() graphql.ID {
	return resolver.id
}

// Name returns the event's name.
func (resolver *PolicyViolationEventResolver) Name() string {
	return resolver.name
}

// Timestamp returns the event's timestamp.
func (resolver *PolicyViolationEventResolver) Timestamp() *graphql.Time {
	return &graphql.Time{Time: resolver.timestamp}
}

func (resolver *Resolver) getPolicyViolationEvents(ctx context.Context, query *v1.Query, predicateFn func(*storage.Alert) bool) ([]*PolicyViolationEventResolver, error) {
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}

	query = paginated.FillDefaultSortOption(query, paginated.GetViolationTimeSortOption())
	alerts, err := resolver.ViolationsDataStore.SearchRawAlerts(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving alerts from search")
	}

	n := 0
	for _, alert := range alerts {
		if predicateFn(alert) {
			alerts[n] = alert
			n++
		}
	}
	alerts = alerts[:n]

	policyViolationEvents := make([]*PolicyViolationEventResolver, 0, len(alerts))
	for _, alert := range alerts {
		timestamp, ok := convertTimestamp(alert.GetPolicy().GetName(), "alert", alert.GetTime())
		if !ok {
			continue
		}
		policy := alert.GetPolicy()
		policyViolationEvents = append(policyViolationEvents, &PolicyViolationEventResolver{
			id:        graphql.ID(policy.GetId()),
			name:      policy.GetName(),
			timestamp: timestamp,
		})
	}

	return policyViolationEvents, nil
}
