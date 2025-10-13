package common

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
)

// MessageInjector is a simplified interface for injecting messages into the central <-> sensor stream.
type MessageInjector interface {
	InjectMessage(ctx concurrency.Waitable, msg *central.MsgToSensor) error
	InjectMessageIntoQueue(msg *central.MsgFromSensor)
}
