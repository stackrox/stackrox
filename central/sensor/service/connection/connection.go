package connection

import (
	"github.com/stackrox/rox/central/scrape"
	"github.com/stackrox/rox/central/sensor/networkpolicies"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
)

// SensorConnection provides a handle to an established connection from a sensor.
type SensorConnection interface {
	common.MessageInjector

	Terminate(err error) bool

	// Stopped returns a signal that, when triggered, guarantees that no more messages from this sensor connection will
	// be processed.
	Stopped() concurrency.ReadOnlyErrorSignal

	Scrapes() scrape.Controller
	NetworkPolicies() networkpolicies.Controller

	ClusterID() string

	InjectMessageIntoQueue(msg *central.MsgFromSensor)
}
