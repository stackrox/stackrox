package common

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
)

func TestSendSensorACK_NACK(t *testing.T) {
	injector := &mockInjector{
		capabilities: map[centralsensor.SensorCapability]bool{
			centralsensor.SensorACKSupport: true,
		},
	}

	SendSensorACK(t.Context(), central.SensorACK_NACK, central.SensorACK_VM_INDEX_REPORT, "vm-nack", centralsensor.SensorACKReasonRateLimited, injector)

	assert.Len(t, injector.messages, 1)
	ack := injector.messages[0].GetSensorAck()
	assert.NotNil(t, ack)
	assert.Equal(t, central.SensorACK_NACK, ack.GetAction())
	assert.Equal(t, central.SensorACK_VM_INDEX_REPORT, ack.GetMessageType())
	assert.Equal(t, "vm-nack", ack.GetResourceId())
	assert.Equal(t, centralsensor.SensorACKReasonRateLimited, ack.GetReason())
}

func TestSendSensorACK_NilInjector(t *testing.T) {
	assert.NotPanics(t, func() {
		SendSensorACK(t.Context(), central.SensorACK_ACK, central.SensorACK_VM_INDEX_REPORT, "vm-1", "", nil)
	})
}

func TestSendSensorACK_InjectorWithoutCapabilitySupport(t *testing.T) {
	injector := &mockInjector{}

	SendSensorACK(t.Context(), central.SensorACK_ACK, central.SensorACK_VM_INDEX_REPORT, "vm-1", "", injector)

	assert.Empty(t, injector.messages, "should not send when SensorACKSupport capability is not advertised")
}

type mockInjector struct {
	messages     []*central.MsgToSensor
	injectErr    error
	capabilities map[centralsensor.SensorCapability]bool
}

func (m *mockInjector) InjectMessage(_ concurrency.Waitable, msg *central.MsgToSensor) error {
	m.messages = append(m.messages, msg)
	return m.injectErr
}

func (m *mockInjector) InjectMessageIntoQueue(_ *central.MsgFromSensor) {}

func (m *mockInjector) HasCapability(cap centralsensor.SensorCapability) bool {
	return m.capabilities[cap]
}
