package event

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestGetEventTypeWithoutPrefix(t *testing.T) {
	var nilDeployment *central.SensorEvent_Deployment = nil
	var nilMap map[string]string = nil
	var nilSlice []string = nil
	var nilStruct any = nil

	tests := map[string]struct {
		input    interface{}
		expected string
	}{
		"should return UnknownEventType for nil input": {
			input:    nil,
			expected: UnknownEventType,
		},
		"should extract type even for typed nil pointer": {
			input:    nilDeployment,
			expected: "Deployment",
		},
		"should extract type even for map nil pointer": {
			input:    nilMap,
			expected: "map[string]string",
		},
		"should extract type even for slice nil pointer": {
			input:    nilSlice,
			expected: "[]string",
		},
		"should return UnknownEventType for any nil pointer": {
			input:    nilStruct,
			expected: UnknownEventType,
		},
		"should extract type for Deployment resource": {
			input:    &central.SensorEvent_Deployment{Deployment: &storage.Deployment{}},
			expected: "Deployment",
		},
		"should extract type for Pod resource": {
			input:    &central.SensorEvent_Pod{Pod: &storage.Pod{}},
			expected: "Pod",
		},
		"should extract type for NetworkPolicy resource": {
			input:    &central.SensorEvent_NetworkPolicy{NetworkPolicy: &storage.NetworkPolicy{}},
			expected: "NetworkPolicy",
		},
		"should extract type for ProcessIndicator resource": {
			input:    &central.SensorEvent_ProcessIndicator{ProcessIndicator: &storage.ProcessIndicator{}},
			expected: "ProcessIndicator",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GetEventTypeWithoutPrefix(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetKeyFromMessage(t *testing.T) {
	tests := map[string]struct {
		msg      *central.MsgFromSensor
		expected string
	}{
		"should generate key with known resource type": {
			msg: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Id: "test-id-123",
						Resource: &central.SensorEvent_Deployment{
							Deployment: &storage.Deployment{},
						},
					},
				},
			},
			expected: "Deployment:test-id-123",
		},
		"should generate key with UnknownEventType for nil resource": {
			msg: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Id:       "test-id-456",
						Resource: nil,
					},
				},
			},
			expected: "UnknownEventType:test-id-456",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GetKeyFromMessage(tt.msg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseIDFromKey(t *testing.T) {
	tests := map[string]struct {
		key      string
		expected string
	}{
		"should extract ID from properly formatted key": {
			key:      "Deployment:test-id-123",
			expected: "test-id-123",
		},
		"should extract ID from key with empty type": {
			key:      ":test-id-456",
			expected: "test-id-456",
		},
		"should return full string when colon not found": {
			key:      "invalid-key",
			expected: "invalid-key",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := ParseIDFromKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatKey(t *testing.T) {
	tests := map[string]struct {
		typ      string
		id       string
		expected string
	}{
		"should format key with type and ID": {
			typ:      "Deployment",
			id:       "test-id-123",
			expected: "Deployment:test-id-123",
		},
		"should format key with empty type": {
			typ:      "",
			id:       "test-id-456",
			expected: ":test-id-456",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := FormatKey(tt.typ, tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}
