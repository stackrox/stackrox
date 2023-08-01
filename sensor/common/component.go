package common

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/message"
	"google.golang.org/grpc"
)

// SensorComponentEvent represents events about which sensor components can be notified
type SensorComponentEvent string

const (
	// SensorComponentEventCentralReachable denotes that Sensor-Central connection is up
	SensorComponentEventCentralReachable SensorComponentEvent = "central-reachable"

	// SensorComponentEventOfflineMode denotes that Sensor-Central connection is broken and sensor should operate in offline mode
	SensorComponentEventOfflineMode SensorComponentEvent = "offline-mode"
)

// Notifiable is the interface used by Sensor to notify components of state changes in Central<->Sensor connectivity.
type Notifiable interface {
	Notify(e SensorComponentEvent)
}

// SensorComponent is one of the components that constitute sensor. It supports for receiving messages from central,
// as well as sending messages back to central.
type SensorComponent interface {
	Notifiable
	Start() error
	Stop(err error) // TODO: get rid of err argument as it always seems to be effectively nil.
	Capabilities() []centralsensor.SensorCapability

	ProcessMessage(msg *central.MsgToSensor) error
	ResponsesC() <-chan *message.ExpiringMessage
}

// MessageToComplianceWithAddress adds the Hostname to sensor.MsgToCompliance so we know where to send it to.
type MessageToComplianceWithAddress struct {
	Msg       *sensor.MsgToCompliance
	Hostname  string
	Broadcast bool
}

// ComplianceComponent is a sensor component that can communicate with compliance. All the messages intended for
// compliance are returned by ComplianceC(). It must be started before the compliance.Multiplexer or we panic.
type ComplianceComponent interface {
	SensorComponent
	Stopped() concurrency.ReadOnlyErrorSignal

	ComplianceC() <-chan MessageToComplianceWithAddress
}

// CentralGRPCConnAware allows to set gRPC connections in sensor components.
// The connection is injected on sensor startup before the Start method gets called.
type CentralGRPCConnAware interface {
	SetCentralGRPCClient(cc grpc.ClientConnInterface)
}
