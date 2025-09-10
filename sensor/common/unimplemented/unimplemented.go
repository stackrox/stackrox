package unimplemented

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
)

// Receiver is a struct intended for components that do not process or handle any messages sent to Sensor.
type Receiver struct{}

func (Receiver) ProcessMessage(_ context.Context, _ *central.MsgToSensor) error {
	return nil
}

func (Receiver) Accepts(_ *central.MsgToSensor) bool {
	return false
}
