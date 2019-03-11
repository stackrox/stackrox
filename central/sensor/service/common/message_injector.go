package common

import "github.com/stackrox/rox/generated/internalapi/central"

// MessageInjector is a simplified interface for injecting messages into the central <-> sensor stream.
type MessageInjector interface {
	InjectMessage(msg *central.MsgToSensor) error
}
