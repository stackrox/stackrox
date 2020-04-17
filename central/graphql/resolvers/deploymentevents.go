package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/processwhitelist"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	processWhitelistPkg "github.com/stackrox/rox/pkg/processwhitelist"
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
			"uid: Int!",
			"parentUid: Int!",
			"whitelisted: Boolean!",
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
	id               graphql.ID
	name             string
	timestamp        time.Time
	uid              int32
	parentUID        int32
	canReadWhitelist bool
	whitelisted      bool
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

// UID returns the process's UID.
func (resolver *ProcessActivityEventResolver) UID() int32 {
	return resolver.uid
}

// ParentUID returns the process's parent's UID.
func (resolver *ProcessActivityEventResolver) ParentUID() int32 {
	return resolver.parentUID
}

// Whitelisted returns true if this process is whitelisted.
func (resolver *ProcessActivityEventResolver) Whitelisted() bool {
	if resolver.canReadWhitelist {
		return resolver.whitelisted
	}
	// Default to true if the requester cannot read the whitelist.
	return true
}

func (resolver *Resolver) getProcessActivityEvents(ctx context.Context, query *v1.Query) ([]*ProcessActivityEventResolver, error) {
	if err := readIndicators(ctx); err != nil {
		return nil, err
	}

	indicators, err := resolver.ProcessIndicatorStore.SearchRawProcessIndicators(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving process indicators from search")
	}

	processEvents := make([]*ProcessActivityEventResolver, 0, len(indicators))
	whitelists := make(map[string]*set.StringSet)
	// This determines if we should read whitelist information.
	// nil means we can.
	canReadWhitelist := readWhitelists(ctx) == nil
	for _, indicator := range indicators {
		var keyStr, procName string
		if canReadWhitelist {
			key := &storage.ProcessWhitelistKey{
				ClusterId:     indicator.GetClusterId(),
				Namespace:     indicator.GetNamespace(),
				DeploymentId:  indicator.GetDeploymentId(),
				ContainerName: indicator.GetContainerName(),
			}
			keyStr = key.String()
			procName = processWhitelistPkg.WhitelistItemFromProcess(indicator)
			if procName != "" {
				if _, exists := whitelists[keyStr]; !exists {
					whitelist, exists, err := resolver.WhiteListDataStore.GetProcessWhitelist(ctx, key)
					if err != nil || !exists {
						log.Error(errors.Wrapf(err, "retrieving whitelist data for process %s", indicator.GetSignal().GetName()))
					} else {
						whitelists[keyStr] = processwhitelist.Processes(whitelist, processwhitelist.RoxOrUserLocked)
					}
				}
			}
		}

		timestamp, ok := convertTimestamp(indicator.GetSignal().GetName(), "indicator", indicator.GetSignal().GetTime())
		if !ok {
			continue
		}
		// -1 indicates we do not have parent UID information (either no parent exists or we do not know its UID).
		parentUID := int32(-1)
		if len(indicator.GetSignal().GetLineageInfo()) > 0 {
			// This process's direct parent should be the first entry.
			parentUID = int32(indicator.GetSignal().GetLineageInfo()[0].GetParentUid())
		}
		processEvents = append(processEvents, &ProcessActivityEventResolver{
			id:               graphql.ID(indicator.GetId()),
			name:             indicator.GetSignal().GetName(),
			timestamp:        timestamp,
			uid:              int32(indicator.GetSignal().GetUid()),
			parentUID:        parentUID,
			canReadWhitelist: canReadWhitelist,
			whitelisted:      whitelists[keyStr] == nil || whitelists[keyStr].Contains(procName),
		})
	}
	return processEvents, nil
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

func (resolver *Resolver) getPolicyViolationEvents(ctx context.Context, query *v1.Query) ([]*PolicyViolationEventResolver, error) {
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}

	alerts, err := resolver.ViolationsDataStore.SearchRawAlerts(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving alerts from search")
	}

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
