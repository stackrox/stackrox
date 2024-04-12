package internalmessage

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

const (
	SensorMessageSoftRestart = iota
)

// SensorInternalMessageCallback is the callback used by subscribers.
type SensorInternalMessageCallback func(message *SensorInternalMessage)

// SensorInternalMessage is the message data structure used by publishers and subscribers to exchange messages.
type SensorInternalMessage struct {
	Kind     int
	Text     string
	Validity context.Context
}

// IsExpired is a helper function that checks if the context already expired without blocking.
// If the context isn't set this function will always return false.
func (im *SensorInternalMessage) IsExpired() bool {

	if im.Validity == nil {
		return false
	}

	select {
	case <-im.Validity.Done():
		return true
	default:
		return false
	}
}

// NewMessageSubscriber creates a MessageSubscriber.
func NewMessageSubscriber() *MessageSubscriber {
	return &MessageSubscriber{
		subscribers: map[int][]SensorInternalMessageCallback{
			SensorMessageSoftRestart: {},
		},
		lock: &sync.RWMutex{},
	}
}

// MessageSubscriber is a lightweight PubSub-like component that can be used to register callbacks and publish
// messages.
type MessageSubscriber struct {
	subscribers map[int][]SensorInternalMessageCallback
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
	return errors.Errorf("message type %d not found: %v", msg.Kind, msg)
}

// Subscribe registers a callback based on a message kind.
func (m *MessageSubscriber) Subscribe(kind int, handler SensorInternalMessageCallback) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if arr, ok := m.subscribers[kind]; ok {
		arr = append(arr, handler)
		m.subscribers[kind] = arr
		return nil
	}
	return errors.Errorf("message type %d not found", kind)
}
