package virtualmachines

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestCapabilities(t *testing.T) {
	pipeline := &pipelineImpl{}
	assert.Nil(t, pipeline.Capabilities())
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name  string
		input *central.MsgFromSensor
		want  bool
	}{
		{
			name:  "nil input",
			input: nil,
			want:  false,
		},
		{
			name:  "empty input",
			input: &central.MsgFromSensor{},
			want:  false,
		},
		{
			name: "bad message type",
			input: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Resource: &central.SensorEvent_Node{
							Node: &storage.Node{
								Id: "node1",
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "match",
			input: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Resource: &central.SensorEvent_VirtualMachine{
							VirtualMachine: &v1.VirtualMachine{
								Id: "virtualMachine1",
							},
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			pipeline := &pipelineImpl{}
			got := pipeline.Match(tt.input)
			assert.Equal(it, tt.want, got)
		})
	}
}
