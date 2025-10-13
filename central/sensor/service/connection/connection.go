package connection

import (
	"github.com/stackrox/rox/central/scrape"
	"github.com/stackrox/rox/central/sensor/networkentities"
	"github.com/stackrox/rox/central/sensor/networkpolicies"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/telemetry"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
)

// SensorConnection provides a handle to an established connection from a sensor.
//
//go:generate mockgen-wrapper
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
