package mocks

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/message"
)

// MockQueue is a mock of the output.Queue interface
type MockQueue struct {
}

// Send implements output.Queue
func (m *MockQueue) Send(_ *message.ResourceEvent) {

}

// ResponseC implements output.Queue
func (m *MockQueue) ResponseC() <-chan *central.MsgFromSensor {
	return nil
}

// NewMockQueue creates a new mock instance
func NewMockQueue() *MockQueue {
	return &MockQueue{}
}
