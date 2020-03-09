package resolvers

import (
	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddInterfaceType("DeploymentEvent", []string{
			"id: ID!",
			"name: String!",
			"timestamp: Int!",
		}),
		schema.AddType("ContainerTerminationEvent", []string{
			"id: ID!",
			"name: String!",
			"timestamp: Int!",
			"exitCode: Int!",
			"reason: String!",
		}, "DeploymentEvent"),
		schema.AddType("ContainerRestartEvent", []string{
			"id: ID!",
			"name: String!",
			"timestamp: Int!",
		}, "DeploymentEvent"),
		schema.AddType("ProcessActivityEvent", []string{
			"id: ID!",
			"name: String!",
			"timestamp: Int!",
			"uid: Int!",
		}, "DeploymentEvent"),
		schema.AddType("PolicyViolationEvent", []string{
			"id: ID!",
			"name: String!",
			"timestamp: Int!",
		}, "DeploymentEvent"),
	)
}

// DeploymentEvent is the parent interface for events.
type DeploymentEvent interface {
	ID() graphql.ID
	Name() string
	Timestamp() int32
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
	timestamp int32
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
func (resolver *ContainerTerminationEventResolver) Timestamp() int32 {
	return resolver.timestamp
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
	timestamp int32
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
func (resolver *ContainerRestartEventResolver) Timestamp() int32 {
	return resolver.timestamp
}

// ProcessActivityEventResolver represents a process start event.
type ProcessActivityEventResolver struct {
	id        graphql.ID
	name      string
	timestamp int32
	uid       uint32
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
func (resolver *ProcessActivityEventResolver) Timestamp() int32 {
	return resolver.timestamp
}

// UID returns the process's UID.
func (resolver *ProcessActivityEventResolver) UID() int32 {
	return int32(resolver.uid)
}

// PolicyViolationEventResolver represents a policy violation event.
type PolicyViolationEventResolver struct {
	id        graphql.ID
	name      string
	timestamp int32
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
func (resolver *PolicyViolationEventResolver) Timestamp() int32 {
	return resolver.timestamp
}
