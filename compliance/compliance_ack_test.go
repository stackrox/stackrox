package compliance

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stretchr/testify/assert"
)

type fakeUMH struct {
	ackCount  int
	nackCount int
}

func (f *fakeUMH) HandleACK()                    { f.ackCount++ }
func (f *fakeUMH) HandleNACK()                   { f.nackCount++ }
func (f *fakeUMH) ObserveSending()               {}
func (f *fakeUMH) RetryCommand() <-chan struct{} { return nil }

func TestHandleNodeScanningComplianceAck(t *testing.T) {
	inv := &fakeUMH{}
	idx := &fakeUMH{}
	c := &Compliance{
		umhNodeInventory: inv,
		umhNodeIndex:     idx,
	}

	tests := []struct {
		name        string
		ack         *sensor.MsgToCompliance_ComplianceACK
		wantInvACK  int
		wantInvNACK int
		wantIdxACK  int
		wantIdxNACK int
	}{
		{
			name: "node inventory ack",
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_ACK,
				MessageType: sensor.MsgToCompliance_ComplianceACK_NODE_INVENTORY,
			},
			wantInvACK: 1,
		},
		{
			name: "node inventory nack",
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_NACK,
				MessageType: sensor.MsgToCompliance_ComplianceACK_NODE_INVENTORY,
			},
			wantInvNACK: 1,
		},
		{
			name: "node index ack",
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_ACK,
				MessageType: sensor.MsgToCompliance_ComplianceACK_NODE_INDEX_REPORT,
			},
			wantIdxACK: 1,
		},
		{
			name: "node index nack",
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_NACK,
				MessageType: sensor.MsgToCompliance_ComplianceACK_NODE_INDEX_REPORT,
			},
			wantIdxNACK: 1,
		},
		{
			name: "vm message type ignored",
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_ACK,
				MessageType: sensor.MsgToCompliance_ComplianceACK_VM_INDEX_REPORT,
			},
		},
		{
			name: "unknown action ignored",
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_Action(999),
				MessageType: sensor.MsgToCompliance_ComplianceACK_NODE_INDEX_REPORT,
			},
		},
		{
			name: "nil message ignored",
			ack:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv.ackCount, inv.nackCount = 0, 0
			idx.ackCount, idx.nackCount = 0, 0
			c.handleNodeScanningComplianceAck(tt.ack)
			assert.Equal(t, tt.wantInvACK, inv.ackCount)
			assert.Equal(t, tt.wantInvNACK, inv.nackCount)
			assert.Equal(t, tt.wantIdxACK, idx.ackCount)
			assert.Equal(t, tt.wantIdxNACK, idx.nackCount)
		})
	}
}
