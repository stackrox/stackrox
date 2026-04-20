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
	ackCount           int
	nackCount          int
	lastACKResourceID  string
	lastNACKResourceID string
	retryC             chan string
}

func (f *fakeUMH) HandleACK(resourceID string) {
	f.ackCount++
	f.lastACKResourceID = resourceID
}

func (f *fakeUMH) HandleNACK(resourceID string) {
	f.nackCount++
	f.lastNACKResourceID = resourceID
}

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

func TestHandleComplianceACK(t *testing.T) {
	inv := &fakeUMH{}
	idx := &fakeUMH{}
	vmIdx := &fakeUMH{}
	c := &Compliance{
		umhNodeInventory: inv,
		umhNodeIndex:     idx,
		umhVMIndex:       vmIdx,
	}

	tests := []struct {
		name         string
		ack          *sensor.MsgToCompliance_ComplianceACK
		wantInvACK   int
		wantInvNACK  int
		wantIdxACK   int
		wantIdxNACK  int
		wantVMIdxACK int
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
			name: "vm index ack",
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_ACK,
				MessageType: sensor.MsgToCompliance_ComplianceACK_VM_INDEX_REPORT,
			},
			wantVMIdxACK: 1,
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
			vmIdx.ackCount, vmIdx.nackCount = 0, 0
			inv.lastACKResourceID, inv.lastNACKResourceID = "", ""
			idx.lastACKResourceID, idx.lastNACKResourceID = "", ""
			vmIdx.lastACKResourceID, vmIdx.lastNACKResourceID = "", ""
			c.handleComplianceACK(tt.ack)
			assert.Equal(t, tt.wantInvACK, inv.ackCount)
			assert.Equal(t, tt.wantInvNACK, inv.nackCount)
			assert.Equal(t, tt.wantIdxACK, idx.ackCount)
			assert.Equal(t, tt.wantIdxNACK, idx.nackCount)
			assert.Equal(t, tt.wantVMIdxACK, vmIdx.ackCount)
		})
	}
}

func TestHandleComplianceACK_VMIndexPairUsesCID(t *testing.T) {
	vmIdx := &fakeUMH{}
	c := &Compliance{
		umhVMIndex: vmIdx,
	}

	c.handleComplianceACK(&sensor.MsgToCompliance_ComplianceACK{
		Action:      sensor.MsgToCompliance_ComplianceACK_ACK,
		MessageType: sensor.MsgToCompliance_ComplianceACK_VM_INDEX_REPORT,
		ResourceId:  "vm-1:100",
	})
	assert.Equalf(t, 1, vmIdx.ackCount, "expected VM index ACK count to be 1, but got %d", vmIdx.ackCount)
	assert.Equalf(t, "100", vmIdx.lastACKResourceID, "expected VM index ACK resource id to be %q, but got %q", "100", vmIdx.lastACKResourceID)

	c.handleComplianceACK(&sensor.MsgToCompliance_ComplianceACK{
		Action:      sensor.MsgToCompliance_ComplianceACK_NACK,
		MessageType: sensor.MsgToCompliance_ComplianceACK_VM_INDEX_REPORT,
		ResourceId:  "vm-2:200",
	})
	assert.Equalf(t, 1, vmIdx.nackCount, "expected VM index NACK count to be 1, but got %d", vmIdx.nackCount)
	assert.Equalf(t, "200", vmIdx.lastNACKResourceID, "expected VM index NACK resource id to be %q, but got %q", "200", vmIdx.lastNACKResourceID)
}

func TestResolveVMRelayResourceID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "returns cid for vmid cid pair",
			input:    "vm-1:123",
			expected: "123",
		},
		{
			name:     "returns unchanged id when no separator is present",
			input:    "123",
			expected: "123",
		},
		{
			name:     "returns unchanged id when vm id is missing",
			input:    ":123",
			expected: ":123",
		},
		{
			name:     "returns unchanged id when cid is missing",
			input:    "vm-1:",
			expected: "vm-1:",
		},
		{
			name:     "returns unchanged id when extra separators are present",
			input:    "vm-1:123:extra",
			expected: "vm-1:123:extra",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := resolveVMRelayResourceID(tc.input)
			assert.Equalf(t, tc.expected, actual, "expected relay resource id %q, but got %q", tc.expected, actual)
		})
	}
}
