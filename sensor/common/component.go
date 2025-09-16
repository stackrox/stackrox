package common

import (
	"context"
	"fmt"

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
	// SensorComponentEventCentralReachable denotes that Sensor-Central gRPC stream is connected and in ready state
	SensorComponentEventCentralReachable SensorComponentEvent = "central-reachable"

	// SensorComponentEventCentralReachableHTTP denotes that Central responds to pings over HTTP
	SensorComponentEventCentralReachableHTTP SensorComponentEvent = "central-reachable-HTTP"

	// SensorComponentEventOfflineMode denotes that Sensor-Central connection is broken and sensor should operate in offline mode
	SensorComponentEventOfflineMode SensorComponentEvent = "offline-mode"

	// SensorComponentEventSyncFinished denotes that Sensor finished initial sync.
	SensorComponentEventSyncFinished SensorComponentEvent = "sync-finished"

	// SensorComponentEventResourceSyncFinished denotes that Sensor finished the k8s resource sync
	SensorComponentEventResourceSyncFinished SensorComponentEvent = "resource-sync-finished"
)

// LogSensorComponentEvent returns a unified string for logging the transition between component states/
// For OfflineAware components, the optional parameter with component name should be provided.
// Variadic `optComponentName` is used for backwards compatibility;
// only first element of `optComponentName` will be printed if more elements are provided.
func LogSensorComponentEvent(e SensorComponentEvent, optComponentName ...string) string {
	name := "Component"
	if len(optComponentName) > 0 {
		name += fmt.Sprintf(" '%s'", optComponentName[0])
	}
	switch e {
	case SensorComponentEventCentralReachable:
		return fmt.Sprintf("%s runs now in Online mode", name)
	case SensorComponentEventOfflineMode:
		return fmt.Sprintf("%s runs now in Offline mode", name)
	case SensorComponentEventSyncFinished:
		return fmt.Sprintf("%s has received the SyncFinished notification", name)
	case SensorComponentEventResourceSyncFinished:
		return fmt.Sprintf("%s has received the ResourceSyncFinished notification", name)
	default:
		return fmt.Sprintf("%s has received the %s notification", name, e)
	}
}

// Notifiable is the interface used by Sensor to notify components of state changes in Central<->Sensor connectivity.
type Notifiable interface {
	Notify(e SensorComponentEvent)
}

// SensorComponent is one of the components that constitute sensor. It supports for receiving messages from central,
// as well as sending messages back to central.
type SensorComponent interface {
	Notifiable
	CentralSender
	CentralReceiver
	Component
}

type Component interface {
	Start() error
	Stop()
	Capabilities() []centralsensor.SensorCapability
	Name() string
}

type CentralSender interface {
	ResponsesC() <-chan *message.ExpiringMessage
}

type CentralReceiver interface {
	// ProcessMessage processes the `msg` message from Central.
	// The `ctx` is used to cancel processing of the message being currently processed.
	ProcessMessage(ctx context.Context, msg *central.MsgToSensor) error
	// Accepts decides weather messages should be processed at all
	Accepts(msg *central.MsgToSensor) bool
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
