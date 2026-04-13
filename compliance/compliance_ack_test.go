package compliance

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
)

// fakeUMH is a minimal test double for node.UnconfirmedMessageHandler.
// Set retryC to a non-nil channel when tests need RetryCommand() to be selectable.
type fakeUMH struct {
	ackCount  int
	nackCount int
	retryC    chan string
}

func (f *fakeUMH) HandleACK(string)      { f.ackCount++ }
func (f *fakeUMH) HandleNACK(string)     { f.nackCount++ }
func (f *fakeUMH) ObserveSending(string) {}
func (f *fakeUMH) OnACK(func(string))    {}

func (f *fakeUMH) RetryCommand() <-chan string {
	if f.retryC != nil {
		return f.retryC
	}
	return nil
}

func (f *fakeUMH) Stopped() concurrency.ReadOnlyErrorSignal {
	s := concurrency.NewStopper()
	s.Flow().ReportStopped()
	return s.Client().Stopped()
}

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
			c.handleComplianceACK(tt.ack)
			assert.Equal(t, tt.wantInvACK, inv.ackCount)
			assert.Equal(t, tt.wantInvNACK, inv.nackCount)
			assert.Equal(t, tt.wantIdxACK, idx.ackCount)
			assert.Equal(t, tt.wantIdxNACK, idx.nackCount)
		})
	}
}
