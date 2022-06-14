package connection

import (
	"github.com/stackrox/stackrox/central/scrape"
	"github.com/stackrox/stackrox/central/sensor/networkentities"
	"github.com/stackrox/stackrox/central/sensor/networkpolicies"
	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/central/sensor/telemetry"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/centralsensor"
	"github.com/stackrox/stackrox/pkg/concurrency"
)

// SensorConnection provides a handle to an established connection from a sensor.
type SensorConnection interface {
	common.MessageInjector

	Terminate(err error) bool

	// Stopped returns a signal that, when triggered, guarantees that no more messages from this sensor connection will
	// be processed.
	Stopped() concurrency.ReadOnlyErrorSignal

	Scrapes() scrape.Controller
	NetworkEntities() networkentities.Controller
	NetworkPolicies() networkpolicies.Controller
	Telemetry() telemetry.Controller

	ClusterID() string

	InjectMessageIntoQueue(msg *central.MsgFromSensor)

	HasCapability(capability centralsensor.SensorCapability) bool

	// ObjectsDeletedByReconciliation returns the count of objects deleted by reconciliation,
	// keyed by type name, as well as a bool returning whether reconciliation has finished.
	ObjectsDeletedByReconciliation() (map[string]int, bool)

	CheckAutoUpgradeSupport() error
}
