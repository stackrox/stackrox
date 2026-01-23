package compliance

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stretchr/testify/suite"
)

func TestCompliance(t *testing.T) {
	suite.Run(t, new(ComplianceTestSuite))
}

type ComplianceTestSuite struct {
	suite.Suite
}

// mockUnconfirmedMessageHandler is a test mock for node.UnconfirmedMessageHandler
type mockUnconfirmedMessageHandler struct {
	ackCount  int
	nackCount int
	retryC    chan string
}

func newMockUnconfirmedMessageHandler() *mockUnconfirmedMessageHandler {
	return &mockUnconfirmedMessageHandler{
		retryC: make(chan string),
	}
}

func (m *mockUnconfirmedMessageHandler) HandleACK(_ string) {
	m.ackCount++
}

func (m *mockUnconfirmedMessageHandler) HandleNACK(_ string) {
	m.nackCount++
}

func (m *mockUnconfirmedMessageHandler) ObserveSending(_ string) {}

func (m *mockUnconfirmedMessageHandler) RetryCommand() <-chan string {
	return m.retryC
}

func (m *mockUnconfirmedMessageHandler) OnACK(_ func(resourceID string)) {
	// no-op for test mock
}

func (s *ComplianceTestSuite) TestHandleComplianceACK() {
	cases := map[string]struct {
		ack                    *sensor.MsgToCompliance_ComplianceACK
		expectedInventoryACKs  int
		expectedInventoryNACKs int
		expectedIndexACKs      int
		expectedIndexNACKs     int
		expectedVMIndexACKs    int
		expectedVMIndexNACKs   int
	}{
		"should handle NODE_INVENTORY ACK": {
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_ACK,
				MessageType: sensor.MsgToCompliance_ComplianceACK_NODE_INVENTORY,
				ResourceId:  "node-1",
			},
			expectedInventoryACKs: 1,
		},
		"should handle NODE_INVENTORY NACK": {
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_NACK,
				MessageType: sensor.MsgToCompliance_ComplianceACK_NODE_INVENTORY,
				ResourceId:  "node-1",
				Reason:      "rate limit exceeded",
			},
			expectedInventoryNACKs: 1,
		},
		"should handle NODE_INDEX_REPORT ACK": {
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_ACK,
				MessageType: sensor.MsgToCompliance_ComplianceACK_NODE_INDEX_REPORT,
				ResourceId:  "node-2",
			},
			expectedIndexACKs: 1,
		},
		"should handle NODE_INDEX_REPORT NACK": {
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_NACK,
				MessageType: sensor.MsgToCompliance_ComplianceACK_NODE_INDEX_REPORT,
				ResourceId:  "node-2",
				Reason:      "central unreachable",
			},
			expectedIndexNACKs: 1,
		},
		"should handle VM_INDEX_REPORT ACK": {
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_ACK,
				MessageType: sensor.MsgToCompliance_ComplianceACK_VM_INDEX_REPORT,
				ResourceId:  "vm-1",
			},
			expectedVMIndexACKs: 1,
		},
		"should handle VM_INDEX_REPORT NACK": {
			ack: &sensor.MsgToCompliance_ComplianceACK{
				Action:      sensor.MsgToCompliance_ComplianceACK_NACK,
				MessageType: sensor.MsgToCompliance_ComplianceACK_VM_INDEX_REPORT,
				ResourceId:  "vm-1",
				Reason:      "rate limit exceeded",
			},
			expectedVMIndexNACKs: 1,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			mockInventory := newMockUnconfirmedMessageHandler()
			mockIndex := newMockUnconfirmedMessageHandler()
			mockVMIndex := newMockUnconfirmedMessageHandler()

			c := &Compliance{
				umhNodeInventory: mockInventory,
				umhNodeIndex:     mockIndex,
				umhVMIndex:       mockVMIndex,
			}

			c.handleComplianceACK(tc.ack)

			s.Equal(tc.expectedInventoryACKs, mockInventory.ackCount, "inventory ACK count")
			s.Equal(tc.expectedInventoryNACKs, mockInventory.nackCount, "inventory NACK count")
			s.Equal(tc.expectedIndexACKs, mockIndex.ackCount, "index ACK count")
			s.Equal(tc.expectedIndexNACKs, mockIndex.nackCount, "index NACK count")
			s.Equal(tc.expectedVMIndexACKs, mockVMIndex.ackCount, "VM index ACK count")
			s.Equal(tc.expectedVMIndexNACKs, mockVMIndex.nackCount, "VM index NACK count")
		})
	}
}

func (s *ComplianceTestSuite) TestHandleComplianceACK_NilACK() {
	mockInventory := newMockUnconfirmedMessageHandler()
	mockIndex := newMockUnconfirmedMessageHandler()
	mockVMIndex := newMockUnconfirmedMessageHandler()

	c := &Compliance{
		umhNodeInventory: mockInventory,
		umhNodeIndex:     mockIndex,
		umhVMIndex:       mockVMIndex,
	}

	// Should not panic and should not call any handlers
	c.handleComplianceACK(nil)

	s.Equal(0, mockInventory.ackCount)
	s.Equal(0, mockInventory.nackCount)
	s.Equal(0, mockIndex.ackCount)
	s.Equal(0, mockIndex.nackCount)
	s.Equal(0, mockVMIndex.ackCount)
	s.Equal(0, mockVMIndex.nackCount)
}
