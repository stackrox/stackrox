package common

import (
	"fmt"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/message"
	"google.golang.org/grpc"
)

// SensorComponentState represents the state of a sensor component
type SensorComponentState int

const (
	// SensorComponentStateUNDEFINED is the state of a sensor component when no state has been reached yet.
	SensorComponentStateUNDEFINED SensorComponentState = 0
	// SensorComponentStateSTARTING is the state of a sensor component which has begun the start process,
	// but not completed it yet.
	SensorComponentStateSTARTING SensorComponentState = 1
	// SensorComponentStateSTARTED is the state of a sensor component which has finished the starting process.
	SensorComponentStateSTARTED SensorComponentState = 2
	// SensorComponentStateONLINE is the state of a sensor component which acknowledged the central reachable event.
	SensorComponentStateONLINE SensorComponentState = 3
	// SensorComponentStateOFFLINE is the state of a sensor component which acknowledged the offline mode event.
	SensorComponentStateOFFLINE SensorComponentState = 4
	// SensorComponentStateSTOPPING is the state of a sensor component which has begun the stop process,
	// but not completed it yet.
	SensorComponentStateSTOPPING SensorComponentState = 5
	// SensorComponentStateSTOPPED is the state of a sensor component which has finished the stopping process.
	SensorComponentStateSTOPPED SensorComponentState = 6
)

func (s *SensorComponentState) String() string {
	if s == nil {
		return "UNKNOWN"
	}
	switch *s {
	case SensorComponentStateUNDEFINED:
		return "undefined"
	case SensorComponentStateSTARTING:
		return "starting"
	case SensorComponentStateSTARTED:
		return "started"
	case SensorComponentStateONLINE:
		return "online"
	case SensorComponentStateOFFLINE:
		return "offline"
	case SensorComponentStateSTOPPING:
		return "stopping"
	case SensorComponentStateSTOPPED:
		return "stopped"
	default:
		return "unknown"
	}
}

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
	Start() error
	Stop(err error) // TODO: get rid of err argument as it always seems to be effectively nil.
	Capabilities() []centralsensor.SensorCapability

	ProcessMessage(msg *central.MsgToSensor) error
	ResponsesC() <-chan *message.ExpiringMessage

	State() SensorComponentState
}

// StateReporter is a function that reports the state of a Sensor component.
type StateReporter func() SensorComponentState

var (
	stateReporterMap = make(map[string]StateReporter)

	stateReporterMapLock = sync.RWMutex{}
)

func RegisterStateReporter(stateSource string, reporter StateReporter) {
	stateReporterMapLock.Lock()
	defer stateReporterMapLock.Unlock()
	stateReporterMap[stateSource] = reporter
}

func GetStateReporters() map[string]StateReporter {
	stateReporterMapLock.RLock()
	defer stateReporterMapLock.RUnlock()
	reporterMap := make(map[string]StateReporter)
	for src, reporter := range stateReporterMap {
		reporterMap[src] = reporter
	}
	return reporterMap
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
