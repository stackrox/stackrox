package message

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
)

type ExpiringMessage struct {
	*central.MsgFromSensor
	Context context.Context
}

func New(msg *central.MsgFromSensor) *ExpiringMessage {
	return NewWithContext(msg, context.Background())
}

func NewWithContext(msg *central.MsgFromSensor, ctx context.Context) *ExpiringMessage {
	expiringMessage := &ExpiringMessage{
		MsgFromSensor: msg,
		Context:       ctx,
	}
	return expiringMessage
}

func (m *ExpiringMessage) Wait() <-chan struct{} {
	if m.Context == nil {
		return context.Background().Done()
	}
	return m.Context.Done()
}

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
