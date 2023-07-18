package message

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
)

// ExpiringMessage is a wrapper on central.MsgFromSensor with the addition of a Context.
// The context will be cancelled when the message is expired and should no longer be sent
// to Central.
type ExpiringMessage struct {
	*central.MsgFromSensor
	Context context.Context
}

// New creates an ExpiringMessage with msg and context.Background.
func New(msg *central.MsgFromSensor) *ExpiringMessage {
	return NewExpiring(context.Background(), msg)
}

// NewExpiring creates a message with a specific context.
func NewExpiring(ctx context.Context, msg *central.MsgFromSensor) *ExpiringMessage {
	return &ExpiringMessage{
		MsgFromSensor: msg,
		Context:       ctx,
	}
}

// Wait is a helper function that will wait until the context on the message is Done.
// This should be used when sending on blocking channels to abort the process the message
// expired in the meantime. If the context is not set, this function blocks forever (i.e. wait on
// context.Background() channel).
func (m *ExpiringMessage) Wait() <-chan struct{} {
	if m.Context == nil {
		return context.Background().Done()
	}
	return m.Context.Done()
}

// IsExpired is a helper function that checks if the context already expired without blocking.
// If the context isn't set this function will always return false.
func (m *ExpiringMessage) IsExpired() bool {
	if m.Context == nil {
		return false
	}

	select {
	case <-m.Context.Done():
		return true
	default:
		return false
	}
}
