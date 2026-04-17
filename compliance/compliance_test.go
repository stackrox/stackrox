package compliance

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/protoassert"
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
			mockInventory := &fakeUMH{retryC: make(chan string)}
			mockIndex := &fakeUMH{retryC: make(chan string)}
			mockVMIndex := &fakeUMH{retryC: make(chan string)}

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
	mockInventory := &fakeUMH{retryC: make(chan string)}
	mockIndex := &fakeUMH{retryC: make(chan string)}
	mockVMIndex := &fakeUMH{retryC: make(chan string)}

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

func TestCheckNodeRelayEligibility(t *testing.T) {
	cases := map[string]struct {
		config                   *sensor.MsgToCompliance_ScrapeConfig
		enableRelayOnMasterNodes string
		expected                 bool
	}{
		"worker node should run relay": {
			config:   &sensor.MsgToCompliance_ScrapeConfig{IsMasterNode: false},
			expected: true,
		},
		"master node should skip relay by default": {
			config:   &sensor.MsgToCompliance_ScrapeConfig{IsMasterNode: true},
			expected: false,
		},
		"master node should run relay when override is enabled": {
			config:                   &sensor.MsgToCompliance_ScrapeConfig{IsMasterNode: true},
			enableRelayOnMasterNodes: "true",
			expected:                 true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.enableRelayOnMasterNodes != "" {
				t.Setenv(env.VirtualMachinesRelayEnabledOnMasterNodes.EnvVar(), tc.enableRelayOnMasterNodes)
			}
			require.Equal(t, tc.expected, shouldStartVMRelay(tc.config))
		})
	}
}

func TestWaitForInitialScrapeConfig(t *testing.T) {
	workerConfig := &sensor.MsgToCompliance_ScrapeConfig{IsMasterNode: false}
	cases := map[string]struct {
		arrange        func(c *Compliance, cancel context.CancelFunc)
		expectedConfig *sensor.MsgToCompliance_ScrapeConfig
	}{
		"should return config after readiness signal": {
			arrange: func(c *Compliance, _ context.CancelFunc) {
				c.scrapeConfig.Store(workerConfig)
				c.scrapeConfigReady.Signal()
			},
			expectedConfig: workerConfig,
		},
		"should return nil when context is cancelled": {
			arrange: func(_ *Compliance, cancel context.CancelFunc) {
				cancel()
			},
			expectedConfig: nil,
		},
		"should return nil when signal fires without config": {
			arrange: func(c *Compliance, _ context.CancelFunc) {
				c.scrapeConfigReady.Signal()
			},
			expectedConfig: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &Compliance{
				scrapeConfigReady: concurrency.NewSignal(),
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			tc.arrange(c, cancel)
			config := c.waitForInitialScrapeConfig(ctx)
			if tc.expectedConfig == nil {
				require.Nil(t, config)
			} else {
				protoassert.Equal(t, tc.expectedConfig, config)
			}
		})
	}
}
