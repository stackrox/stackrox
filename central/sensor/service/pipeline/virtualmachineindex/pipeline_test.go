package virtualmachineindex

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/connection"
	connMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	vmDatastoreMocks "github.com/stackrox/rox/central/virtualmachine/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/rate"
	"github.com/stackrox/rox/pkg/sync"
	vmEnricherMocks "github.com/stackrox/rox/pkg/virtualmachine/enricher/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	testClusterID = "test-cluster-id"
)

var ctx = context.Background()

// mustNewLimiter creates a rate limiter or fails the test.
func mustNewLimiter(t require.TestingT, workloadName string, globalRate float64, bucketCapacity int) *rate.Limiter {
	limiter, err := rate.NewLimiter(workloadName, globalRate, bucketCapacity)
	require.NoError(t, err)
	return limiter
}

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	vmDatastore *vmDatastoreMocks.MockDataStore
	enricher    *vmEnricherMocks.MockVirtualMachineEnricher
	pipeline    *pipelineImpl

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.vmDatastore = vmDatastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.enricher = vmEnricherMocks.NewMockVirtualMachineEnricher(suite.mockCtrl)
	// Use unlimited rate limiter for tests (rate=0)
	rateLimiter := mustNewLimiter(suite.T(), "test", 0, 50)
	suite.pipeline = &pipelineImpl{
		vmDatastore: suite.vmDatastore,
		enricher:    suite.enricher,
		rateLimiter: rateLimiter,
	}
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

// Helper function to create a virtual machine message
func createVMIndexMessage(vmID string, action central.ResourceAction) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     vmID,
				Action: action,
				Resource: &central.SensorEvent_VirtualMachineIndexReport{
					VirtualMachineIndexReport: &v1.IndexReportEvent{
						Id: vmID,
						Index: &v1.IndexReport{
							IndexV4: &v4.IndexReport{
								Contents: &v4.Contents{
									Packages: map[string]*v4.Package{
										"pkg-1": {
											Id:      "pkg-1",
											Name:    "test-package",
											Version: "1.0.0",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Helper function to create a non-VM message
func createNonVMMessage() *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     "test-id",
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_Node{
					Node: &storage.Node{
						Id:   "node-id",
						Name: "node-name",
					},
				},
			},
		},
	}
}

func (suite *PipelineTestSuite) TestMatch_VirtualMachineMessage() {
	msg := createVMIndexMessage("vm-1", central.ResourceAction_SYNC_RESOURCE)
	result := suite.pipeline.Match(msg)
	suite.True(result, "Should match virtual machine messages")
}

func (suite *PipelineTestSuite) TestRun_NilVirtualMachine() {
	suite.T().Setenv(features.VirtualMachines.EnvVar(), "true")
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     "test-id",
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: nil,
				},
			},
		},
	}

	err := suite.pipeline.Run(ctx, testClusterID, msg, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "unexpected resource type")
}

func (suite *PipelineTestSuite) TestRun_UpdateScanError() {
	suite.T().Setenv(features.VirtualMachines.EnvVar(), "true")
	vmID := "vm-1"
	msg := createVMIndexMessage(vmID, central.ResourceAction_SYNC_RESOURCE)

	// Expect enricher to be called successfully
	suite.enricher.EXPECT().
		EnrichVirtualMachineWithVulnerabilities(gomock.Any(), gomock.Any()).
		Return(nil)

	expectedError := errors.New("datastore error")
	suite.vmDatastore.EXPECT().
		UpdateVirtualMachineScan(ctx, vmID, gomock.Any()).
		Return(expectedError)

	err := suite.pipeline.Run(ctx, testClusterID, msg, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "failed to upsert VM vm-1 to datastore: datastore error")
	suite.Contains(err.Error(), "datastore error")
}

func (suite *PipelineTestSuite) TestCapabilities() {
	capabilities := suite.pipeline.Capabilities()
	suite.Contains(capabilities, centralsensor.CentralCapability(centralsensor.VirtualMachinesSupported))
}

func (suite *PipelineTestSuite) TestOnFinish() {
	// OnFinish should not panic and should be a no-op
	suite.NotPanics(func() {
		suite.pipeline.OnFinish(testClusterID)
	})
}

func (suite *PipelineTestSuite) TestReconcile() {
	// Reconcile should be a no-op and return nil
	storeMap := reconciliation.NewStoreMap()
	err := suite.pipeline.Reconcile(ctx, testClusterID, storeMap)
	suite.NoError(err)
}

// Test the factory functions
func (suite *PipelineTestSuite) TestGetPipeline() {
	pipeline := GetPipeline()
	suite.NotNil(pipeline)
	suite.IsType(&pipelineImpl{}, pipeline)
}

func (suite *PipelineTestSuite) TestNewPipeline() {
	mockDatastore := vmDatastoreMocks.NewMockDataStore(suite.mockCtrl)
	mockEnricher := vmEnricherMocks.NewMockVirtualMachineEnricher(suite.mockCtrl)
	rateLimiter := mustNewLimiter(suite.T(), "test", 0, 50)
	pipeline := newPipeline(mockDatastore, mockEnricher, rateLimiter)
	suite.NotNil(pipeline)

	impl, ok := pipeline.(*pipelineImpl)
	suite.True(ok, "Should return pipelineImpl instance")
	suite.Equal(mockDatastore, impl.vmDatastore)
	suite.Equal(mockEnricher, impl.enricher)
	suite.Equal(rateLimiter, impl.rateLimiter)
}

// Test table-driven approach for different actions
func TestPipelineRun_DifferentActions(t *testing.T) {
	tests := []struct {
		name          string
		action        central.ResourceAction
		expectUpdate  bool
		expectError   bool
		errorContains string
	}{
		{
			name:         "CREATE_RESOURCE",
			action:       central.ResourceAction_CREATE_RESOURCE,
			expectUpdate: false,
		},
		{
			name:         "UPDATE_RESOURCE",
			action:       central.ResourceAction_UPDATE_RESOURCE,
			expectUpdate: false,
		},
		{
			name:         "UNSET_ACTION_RESOURCE",
			action:       central.ResourceAction_UNSET_ACTION_RESOURCE,
			expectUpdate: false,
		},
		{
			name:         "REMOVE_RESOURCE",
			action:       central.ResourceAction_REMOVE_RESOURCE,
			expectUpdate: false,
		},
		{
			name:         "SYNC_RESOURCE",
			action:       central.ResourceAction_SYNC_RESOURCE,
			expectUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(features.VirtualMachines.EnvVar(), "true")
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			vmDatastore := vmDatastoreMocks.NewMockDataStore(ctrl)
			enricher := vmEnricherMocks.NewMockVirtualMachineEnricher(ctrl)
			rateLimiter := mustNewLimiter(t, "test", 0, 50)
			pipeline := &pipelineImpl{
				vmDatastore: vmDatastore,
				enricher:    enricher,
				rateLimiter: rateLimiter,
			}

			vmID := "vm-1"
			msg := createVMIndexMessage(vmID, tt.action)

			if tt.expectUpdate {
				// Expect enricher to be called for action
				enricher.EXPECT().
					EnrichVirtualMachineWithVulnerabilities(gomock.Any(), gomock.Any()).
					Return(nil)

				vmDatastore.EXPECT().
					UpdateVirtualMachineScan(ctx, vmID, gomock.Any()).
					Do(func(ctx context.Context, virtualMachineID string, _ *storage.VirtualMachineScan) {
						assert.Equal(t, vmID, virtualMachineID)
					}).
					Return(nil)
			}

			err := pipeline.Run(ctx, testClusterID, msg, nil)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test edge cases with malformed messages
func TestPipelineEdgeCases(t *testing.T) {
	t.Setenv(features.VirtualMachines.EnvVar(), "true")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vmDatastore := vmDatastoreMocks.NewMockDataStore(ctrl)
	rateLimiter := mustNewLimiter(t, "test", 0, 50)
	pipeline := &pipelineImpl{
		vmDatastore: vmDatastore,
		rateLimiter: rateLimiter,
	}

	t.Run("nil message", func(t *testing.T) {
		result := pipeline.Match(nil)
		assert.False(t, result)
	})

	t.Run("message with nil event", func(t *testing.T) {
		msg := &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_Event{
				Event: nil,
			},
		}
		result := pipeline.Match(msg)
		assert.False(t, result)
	})

	t.Run("message with wrong event type", func(t *testing.T) {
		msg := createNonVMMessage()
		result := pipeline.Match(msg)
		assert.False(t, result, "Should not match non-virtual machine messages")
	})

	t.Run("message with sensorHello", func(t *testing.T) {
		msg := &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_Hello{
				Hello: &central.SensorHello{},
			},
		}
		result := pipeline.Match(msg)
		assert.False(t, result, "Should not match messages without events")
	})

	t.Run("event with wrong resource type", func(t *testing.T) {
		msg := &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Resource: &central.SensorEvent_Node{
						Node: &storage.Node{},
					},
				},
			},
		}
		err := pipeline.Run(ctx, testClusterID, msg, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected resource type")
	})
}

func TestPipelineRun_DisabledFeature(t *testing.T) {
	t.Setenv(features.VirtualMachines.EnvVar(), "false")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vmDatastore := vmDatastoreMocks.NewMockDataStore(ctrl)
	enricher := vmEnricherMocks.NewMockVirtualMachineEnricher(ctrl)
	rateLimiter := mustNewLimiter(t, "test", 0, 50)
	pipeline := &pipelineImpl{
		vmDatastore: vmDatastore,
		enricher:    enricher,
		rateLimiter: rateLimiter,
	}

	vmID := "vm-1"
	msg := createVMIndexMessage(vmID, central.ResourceAction_CREATE_RESOURCE)

	err := pipeline.Run(ctx, testClusterID, msg, nil)

	assert.NoError(t, err)
}

// TestPipelineRun_RateLimitDisabled tests that rate limiting is disabled when configured with 0
func TestPipelineRun_RateLimitDisabled(t *testing.T) {
	t.Setenv(features.VirtualMachines.EnvVar(), "true")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vmDatastore := vmDatastoreMocks.NewMockDataStore(ctrl)
	enricher := vmEnricherMocks.NewMockVirtualMachineEnricher(ctrl)
	rateLimiter := mustNewLimiter(t, "test", 0, 50) // Disabled

	pipeline := &pipelineImpl{
		vmDatastore: vmDatastore,
		enricher:    enricher,
		rateLimiter: rateLimiter,
	}

	vmID := "vm-1"
	msg := createVMIndexMessage(vmID, central.ResourceAction_SYNC_RESOURCE)

	// Should process all 100 requests without rate limiting
	for i := 0; i < 100; i++ {
		enricher.EXPECT().
			EnrichVirtualMachineWithVulnerabilities(gomock.Any(), gomock.Any()).
			Return(nil)
		vmDatastore.EXPECT().
			UpdateVirtualMachineScan(ctx, vmID, gomock.Any()).
			Return(nil)

		err := pipeline.Run(ctx, testClusterID, msg, nil)
		assert.NoError(t, err, "request %d should succeed with rate limiting disabled", i)
	}
}

// TestPipelineRun_RateLimitEnabled tests that rate limiting rejects requests when enabled.
// This test verifies that:
// 1. First N requests (within burst) succeed and perform enrichment/datastore writes
// 2. Rate-limited request does NOT perform enrichment or datastore writes
// 3. A NACK is sent for rate-limited requests when ACK support is enabled
func TestPipelineRun_RateLimitEnabled(t *testing.T) {
	t.Setenv(features.VirtualMachines.EnvVar(), "true")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vmDatastore := vmDatastoreMocks.NewMockDataStore(ctrl)
	enricher := vmEnricherMocks.NewMockVirtualMachineEnricher(ctrl)
	rateLimiter := mustNewLimiter(t, "test", 5, 5) // 5 req/s, bucket capacity=5

	// Recording injector to capture sent messages
	injector := &recordingInjector{}

	// Mock connection with SensorACKSupport capability
	mockConn := connMocks.NewMockSensorConnection(ctrl)
	mockConn.EXPECT().HasCapability(centralsensor.SensorACKSupport).Return(true).AnyTimes()

	pipeline := &pipelineImpl{
		vmDatastore: vmDatastore,
		enricher:    enricher,
		rateLimiter: rateLimiter,
	}

	vmID := "vm-1"
	msg := createVMIndexMessage(vmID, central.ResourceAction_SYNC_RESOURCE)

	// Build a context with the mocked connection that has SensorACKSupport
	ctxWithConn := connection.WithConnection(context.Background(), mockConn)

	// Expect enrichment and datastore writes ONLY for the first 5 (non-rate-limited) requests.
	// The 6th request should be rate-limited and these methods should NOT be called.
	enricher.EXPECT().
		EnrichVirtualMachineWithVulnerabilities(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(5)

	vmDatastore.EXPECT().
		UpdateVirtualMachineScan(gomock.Any(), vmID, gomock.Any()).
		Return(nil).
		Times(5)

	// Send 6 requests - the first 5 should be processed successfully,
	// the 6th should be rate-limited.
	for i := 0; i < 6; i++ {
		err := pipeline.Run(ctxWithConn, testClusterID, msg, injector)
		assert.NoError(t, err, "Run should not return an error even when rate-limited (request %d)", i+1)
	}

	// Verify ACKs were sent for successful requests and NACK for rate-limited request
	acks := injector.getSentACKs()
	require.Len(t, acks, 6, "expected 6 ACK/NACK messages (5 ACKs + 1 NACK)")

	// First 5 should be ACKs
	for i := range 5 {
		assert.Equal(t, central.SensorACK_ACK, acks[i].GetAction(), "request %d should be ACKed", i+1)
		assert.Equal(t, central.SensorACK_VM_INDEX_REPORT, acks[i].GetMessageType())
	}

	// 6th should be NACK
	assert.Equal(t, central.SensorACK_NACK, acks[5].GetAction(), "request 6 should be NACKed (rate limited)")
	assert.Equal(t, central.SensorACK_VM_INDEX_REPORT, acks[5].GetMessageType())
	assert.Contains(t, acks[5].GetReason(), "rate limit exceeded")
}

// TestPipelineRun_NilRateLimiter_WithACKSupport tests behavior when the rateLimiter is nil and ACKs are supported.
// This covers the nil-limiter branch and verifies that:
// 1. No enrichment/datastore calls occur
// 2. A NACK with MessageType=VM_INDEX_REPORT is sent
func TestPipelineRun_NilRateLimiter_WithACKSupport(t *testing.T) {
	t.Setenv(features.VirtualMachines.EnvVar(), "true")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mocks for datastore and enricher - no expectations should be set on these,
	// because the pipeline must short-circuit before doing any work.
	vmDatastore := vmDatastoreMocks.NewMockDataStore(ctrl)
	enricher := vmEnricherMocks.NewMockVirtualMachineEnricher(ctrl)

	// Recording injector to capture sent messages
	injector := &recordingInjector{}

	// Mock connection with SensorACKSupport capability
	mockConn := connMocks.NewMockSensorConnection(ctrl)
	mockConn.EXPECT().HasCapability(centralsensor.SensorACKSupport).Return(true).AnyTimes()

	pipeline := &pipelineImpl{
		vmDatastore: vmDatastore,
		enricher:    enricher,
		rateLimiter: nil, // nil rate limiter to cover the nil-limiter branch
	}

	vmID := "vm-1"
	msg := createVMIndexMessage(vmID, central.ResourceAction_SYNC_RESOURCE)

	// Build a context with the mocked connection that has SensorACKSupport
	ctxWithConn := connection.WithConnection(context.Background(), mockConn)

	// Run the pipeline - it should short-circuit due to nil rateLimiter,
	// emit a NACK, and not call any datastore/enricher methods.
	err := pipeline.Run(ctxWithConn, testClusterID, msg, injector)
	assert.NoError(t, err, "pipeline Run should not error when rateLimiter is nil")

	// Verify exactly one NACK was sent
	acks := injector.getSentACKs()
	require.Len(t, acks, 1, "expected exactly one ACK/NACK to be sent")

	ack := acks[0]
	assert.Equal(t, central.SensorACK_NACK, ack.GetAction(), "expected NACK action")
	assert.Equal(t, central.SensorACK_VM_INDEX_REPORT, ack.GetMessageType(), "expected VM_INDEX_REPORT message type")
	assert.Equal(t, vmID, ack.GetResourceId(), "expected resource ID to match VM ID")
	assert.Equal(t, "nil rate limiter", ack.GetReason(), "expected reason to indicate nil rate limiter")
}

// recordingInjector is a test double that records all SensorACK messages sent via InjectMessage.
var _ common.MessageInjector = (*recordingInjector)(nil)

type recordingInjector struct {
	lock     sync.Mutex
	messages []*central.SensorACK
}

func (r *recordingInjector) InjectMessage(_ concurrency.Waitable, msg *central.MsgToSensor) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if ack := msg.GetSensorAck(); ack != nil {
		r.messages = append(r.messages, ack.CloneVT())
	}
	return nil
}

func (r *recordingInjector) InjectMessageIntoQueue(_ *central.MsgFromSensor) {}

func (r *recordingInjector) getSentACKs() []*central.SensorACK {
	r.lock.Lock()
	defer r.lock.Unlock()
	copied := make([]*central.SensorACK, 0, len(r.messages))
	copied = append(copied, r.messages...)
	return copied
}

// TestOnFinishPropagatesClusterDisconnect verifies that OnFinish propagates the cluster ID
// to the rate limiter's OnClientDisconnect method.
func TestOnFinishPropagatesClusterDisconnect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vmDatastore := vmDatastoreMocks.NewMockDataStore(ctrl)
	enricher := vmEnricherMocks.NewMockVirtualMachineEnricher(ctrl)

	// Use a fake limiter so we can observe calls to OnClientDisconnect.
	fakeLimiter := &fakeRateLimiter{}

	p := &pipelineImpl{
		vmDatastore: vmDatastore,
		enricher:    enricher,
		rateLimiter: fakeLimiter,
	}

	const clusterID = "cluster-1"

	p.OnFinish(clusterID)

	assert.Equal(t, clusterID, fakeLimiter.lastDisconnectedClientID, "OnFinish should propagate cluster disconnect to the rate limiter")
}

// TestOnFinishWithNilRateLimiter verifies that OnFinish doesn't panic when rateLimiter is nil.
func TestOnFinishWithNilRateLimiter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vmDatastore := vmDatastoreMocks.NewMockDataStore(ctrl)
	enricher := vmEnricherMocks.NewMockVirtualMachineEnricher(ctrl)

	p := &pipelineImpl{
		vmDatastore: vmDatastore,
		enricher:    enricher,
		rateLimiter: nil,
	}

	// Should not panic
	assert.NotPanics(t, func() {
		p.OnFinish("cluster-1")
	})
}

// fakeRateLimiter is a test double that records the last client ID passed to OnClientDisconnect.
// It satisfies the interface used by pipelineImpl.rateLimiter.
type fakeRateLimiter struct {
	lastDisconnectedClientID string
}

func (f *fakeRateLimiter) TryConsume(_ string) (bool, string) {
	return true, ""
}

func (f *fakeRateLimiter) OnClientDisconnect(clientID string) {
	f.lastDisconnectedClientID = clientID
}
