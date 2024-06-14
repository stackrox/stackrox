package internalmessage

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	// SensorMessageSoftRestart is a message kind where components require sensor-central connection to restart.
	SensorMessageSoftRestart = "SensorMessage_SoftRestart"

	// SensorMessageResourceSyncFinished is a message kind where components require the resource sync to be finished.
	SensorMessageResourceSyncFinished = "SensorMessage_ResourceSyncFinished"
)

// SensorInternalMessageCallback is the callback used by subscribers.
type SensorInternalMessageCallback func(message *SensorInternalMessage)

// NewMessageSubscriber creates a MessageSubscriber.
func NewMessageSubscriber() *MessageSubscriber {
	return &MessageSubscriber{
		subscribers: map[string][]SensorInternalMessageCallback{
			SensorMessageSoftRestart:          {},
			SensorMessageResourceSyncFinished: {},
		},
		lock: &sync.RWMutex{},
	}
}

// MessageSubscriber is a lightweight PubSub-like component that can be used to register callbacks and publish
// messages.
type MessageSubscriber struct {
	subscribers map[string][]SensorInternalMessageCallback
	lock        *sync.RWMutex
}

// Publish a message to all subscribers.
func (m *MessageSubscriber) Publish(msg *SensorInternalMessage) error {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if arr, ok := m.subscribers[msg.Kind]; ok {
		for _, subCallback := range arr {
			go subCallback(msg)
		}
		return nil
	}
	return errors.Errorf("message type %s not found: %v", msg.Kind, msg)
}

// Subscribe registers a callback based on a message kind.
func (m *MessageSubscriber) Subscribe(kind string, handler SensorInternalMessageCallback) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if arr, ok := m.subscribers[kind]; ok {
		arr = append(arr, handler)
		m.subscribers[kind] = arr
		return nil
	}
	return errors.Errorf("message type %s not found", kind)
}
