package common

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"google.golang.org/grpc"
)

// SensorComponentEvent represents events about which sensor components can be notified
type SensorComponentEvent string

const (
	// SensorComponentEventCentralReachable denotes that Sensor-Central connection is up
	SensorComponentEventCentralReachable SensorComponentEvent = "central-reachable"
)

// SensorComponent is one of the components that constitute sensor. It supports for receiving messages from central,
// as well as sending messages back to central.
type SensorComponent interface {
	Start() error
	Stop(err error) // TODO: get rid of err argument as it always seems to be effectively nil.
	Notify(e SensorComponentEvent)
	Capabilities() []centralsensor.SensorCapability

	ProcessMessage(msg *central.MsgToSensor) error
	ResponsesC() <-chan *central.MsgFromSensor
}

// CentralGRPCConnAware allows to set gRPC connections in sensor components.
// The connection is injected on sensor startup before the Start method gets called.
type CentralGRPCConnAware interface {
	SetCentralGRPCClient(cc grpc.ClientConnInterface)
}
