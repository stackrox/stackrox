package common

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"google.golang.org/grpc"
)

// SensorComponent is one of the components that constitute sensor. It supports for receiving messages from central,
// as well as sending messages back to central.
type SensorComponent interface {
	Start() error
	Stop(err error)
	Capabilities() []centralsensor.SensorCapability

	ProcessMessage(msg *central.MsgToSensor) error
	ResponsesC() <-chan *central.MsgFromSensor
}

// CentralGRPCConnAware allows to set gRPC connections in sensor components.
// The connection is injected on sensor startup before the Start method gets called.
type CentralGRPCConnAware interface {
	SetCentralGRPCClient(cc grpc.ClientConnInterface)
}
