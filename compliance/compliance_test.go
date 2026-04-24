package compliance

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
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

func TestWaitForVMRelayEligibility(t *testing.T) {
	t.Run("should start for nil config as safe fallback", func(t *testing.T) {
		configC := make(chan *sensor.MsgToCompliance_ScrapeConfig, 1)
		defer close(configC)
		configC <- (*sensor.MsgToCompliance_ScrapeConfig)(nil)

		require.True(t, waitForVMRelayEligibility(context.Background(), configC))
	})

	t.Run("should start immediately for worker config", func(t *testing.T) {
		t.Setenv(env.VirtualMachinesRelayEnabledOnMasterNodes.EnvVar(), "")
		configC := make(chan *sensor.MsgToCompliance_ScrapeConfig, 1)
		defer close(configC)
		configC <- &sensor.MsgToCompliance_ScrapeConfig{IsMasterNode: false}

		require.True(t, waitForVMRelayEligibility(context.Background(), configC))
	})

	t.Run("should wait for later eligible config", func(t *testing.T) {
		t.Setenv(env.VirtualMachinesRelayEnabledOnMasterNodes.EnvVar(), "")
		configC := make(chan *sensor.MsgToCompliance_ScrapeConfig, 2)
		defer close(configC)
		configC <- &sensor.MsgToCompliance_ScrapeConfig{IsMasterNode: true}
		configC <- &sensor.MsgToCompliance_ScrapeConfig{IsMasterNode: false}

		require.True(t, waitForVMRelayEligibility(context.Background(), configC))
	})

	t.Run("should return false when channel closes before eligibility", func(t *testing.T) {
		t.Setenv(env.VirtualMachinesRelayEnabledOnMasterNodes.EnvVar(), "")
		configC := make(chan *sensor.MsgToCompliance_ScrapeConfig, 1)
		configC <- &sensor.MsgToCompliance_ScrapeConfig{IsMasterNode: true}
		close(configC)

		require.False(t, waitForVMRelayEligibility(context.Background(), configC))
	})

	t.Run("should return false when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		configC := make(chan *sensor.MsgToCompliance_ScrapeConfig)
		defer close(configC)

		require.False(t, waitForVMRelayEligibility(ctx, configC))
	})
}

func TestPublishLatestVMRelayConfig(t *testing.T) {
	t.Run("should publish when channel is empty", func(t *testing.T) {
		configC := make(chan *sensor.MsgToCompliance_ScrapeConfig, 1)
		defer close(configC)
		config := &sensor.MsgToCompliance_ScrapeConfig{IsMasterNode: false}

		publishLatestVMRelayConfig(configC, config)
		protoassert.Equal(t, config, <-configC)
	})

	t.Run("should replace stale config when channel is full", func(t *testing.T) {
		configC := make(chan *sensor.MsgToCompliance_ScrapeConfig, 1)
		defer close(configC)
		staleConfig := &sensor.MsgToCompliance_ScrapeConfig{IsMasterNode: true}
		latestConfig := &sensor.MsgToCompliance_ScrapeConfig{IsMasterNode: false}

		configC <- staleConfig
		publishLatestVMRelayConfig(configC, latestConfig)

		protoassert.Equal(t, latestConfig, <-configC)
	})
}
