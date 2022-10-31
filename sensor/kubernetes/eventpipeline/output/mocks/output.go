package mocks

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/message"
)

// MockQueue is a mock of the output.Queue interface
type MockQueue struct {
}

// Send implements OutputQueue
func (m *MockQueue) Send(_ *message.ResourceEvent) {

}

// ResponsesC implements OutputQueue
func (m *MockQueue) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}

// NewMockQueue creates a new mock instance
func NewMockQueue() *MockQueue {
	return &MockQueue{}
}

// Start implements OutputQueue
func (m *MockQueue) Start() error {
	return nil
}

// Stop implements OutputQueue
func (m *MockQueue) Stop(_ error) {

}
