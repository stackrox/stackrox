package compliance

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestCompliance(t *testing.T) {
	suite.Run(t, new(ComplianceTestSuite))
}

type ComplianceTestSuite struct {
	suite.Suite
}

func (s *ComplianceTestSuite) TestHandleComplianceACK() {
	cases := map[string]struct {
		ack                    *sensor.MsgToCompliance_ComplianceACK
		expectedInventoryACKs  int
		expectedInventoryNACKs int
		expectedIndexACKs      int
		expectedIndexNACKs     int
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
	}

	for name, tc := range cases {
		s.Run(name, func() {
			mockInventory := &fakeUMH{retryC: make(chan string)}
			mockIndex := &fakeUMH{retryC: make(chan string)}

			c := &Compliance{
				umhNodeInventory: mockInventory,
				umhNodeIndex:     mockIndex,
			}

			c.handleComplianceACK(tc.ack)

			s.Equal(tc.expectedInventoryACKs, mockInventory.ackCount, "inventory ACK count")
			s.Equal(tc.expectedInventoryNACKs, mockInventory.nackCount, "inventory NACK count")
			s.Equal(tc.expectedIndexACKs, mockIndex.ackCount, "index ACK count")
			s.Equal(tc.expectedIndexNACKs, mockIndex.nackCount, "index NACK count")
		})
	}
}

func (s *ComplianceTestSuite) TestHandleComplianceACK_NilACK() {
	mockInventory := &fakeUMH{retryC: make(chan string)}
	mockIndex := &fakeUMH{retryC: make(chan string)}

	c := &Compliance{
		umhNodeInventory: mockInventory,
		umhNodeIndex:     mockIndex,
	}

	// Should not panic and should not call any handlers
	c.handleComplianceACK(nil)

	s.Equal(0, mockInventory.ackCount)
	s.Equal(0, mockInventory.nackCount)
	s.Equal(0, mockIndex.ackCount)
	s.Equal(0, mockIndex.nackCount)
}

func TestShouldRunVMRelay(t *testing.T) {
	cases := map[string]struct {
		config                   *sensor.MsgToCompliance_ScrapeConfig
		enableRelayOnMasterNodes string
		expectedShouldRunVMRelay bool
	}{
		"nil config should run relay for safety": {
			config:                   nil,
			enableRelayOnMasterNodes: "",
			expectedShouldRunVMRelay: true,
		},
		"worker node should run relay": {
			config: &sensor.MsgToCompliance_ScrapeConfig{
				IsMasterNode: false,
			},
			enableRelayOnMasterNodes: "",
			expectedShouldRunVMRelay: true,
		},
		"master node should skip relay by default": {
			config: &sensor.MsgToCompliance_ScrapeConfig{
				IsMasterNode: true,
			},
			enableRelayOnMasterNodes: "",
			expectedShouldRunVMRelay: false,
		},
		"master node should run relay when override is enabled": {
			config: &sensor.MsgToCompliance_ScrapeConfig{
				IsMasterNode: true,
			},
			enableRelayOnMasterNodes: "true",
			expectedShouldRunVMRelay: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env.VirtualMachinesRelayEnabledOnMasterNodes.EnvVar(), tc.enableRelayOnMasterNodes)
			require.Equal(t, tc.expectedShouldRunVMRelay, shouldRunVMRelay(tc.config))
		})
	}
}
