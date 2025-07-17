package unimplemented

import "github.com/stackrox/rox/generated/internalapi/central"

type Receiver struct{}

func (Receiver) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}
