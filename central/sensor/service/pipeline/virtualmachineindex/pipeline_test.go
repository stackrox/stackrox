package virtualmachineindex

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	vmDatastoreMocks "github.com/stackrox/rox/central/virtualmachine/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	vmEnricherMocks "github.com/stackrox/rox/pkg/virtualmachine/enricher/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	testClusterID = "test-cluster-id"
)

var ctx = context.Background()

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
	suite.pipeline = &pipelineImpl{
		vmDatastore: suite.vmDatastore,
		enricher:    suite.enricher,
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
	pipeline := newPipeline(mockDatastore, mockEnricher)
	suite.NotNil(pipeline)

	impl, ok := pipeline.(*pipelineImpl)
	suite.True(ok, "Should return pipelineImpl instance")
	suite.Equal(mockDatastore, impl.vmDatastore)
	suite.Equal(mockEnricher, impl.enricher)
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
			pipeline := &pipelineImpl{
				vmDatastore: vmDatastore,
				enricher:    enricher,
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
	pipeline := &pipelineImpl{
		vmDatastore: vmDatastore,
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

// mockInjector records InjectMessage calls.
type mockInjector struct {
	messages  []*central.MsgToSensor
	injectErr error
}

func (m *mockInjector) InjectMessage(_ concurrency.Waitable, msg *central.MsgToSensor) error {
	m.messages = append(m.messages, msg)
	return m.injectErr
}

func (m *mockInjector) InjectMessageIntoQueue(_ *central.MsgFromSensor) {}

// mockInjectorWithCaps wraps mockInjector and additionally implements capabilityChecker.
type mockInjectorWithCaps struct {
	mockInjector
	capabilities map[centralsensor.SensorCapability]bool
}

func (m *mockInjectorWithCaps) HasCapability(cap centralsensor.SensorCapability) bool {
	return m.capabilities[cap]
}

func (suite *PipelineTestSuite) TestRun_SendsACKOnSuccess() {
	suite.T().Setenv(features.VirtualMachines.EnvVar(), "true")
	vmID := "vm-ack-test"
	msg := createVMIndexMessage(vmID, central.ResourceAction_SYNC_RESOURCE)

	suite.enricher.EXPECT().
		EnrichVirtualMachineWithVulnerabilities(gomock.Any(), gomock.Any()).
		Return(nil)
	suite.vmDatastore.EXPECT().
		UpdateVirtualMachineScan(ctx, vmID, gomock.Any()).
		Return(nil)

	injector := &mockInjectorWithCaps{
		capabilities: map[centralsensor.SensorCapability]bool{
			centralsensor.SensorACKSupport: true,
		},
	}

	err := suite.pipeline.Run(ctx, testClusterID, msg, injector)
	suite.NoError(err)

	suite.Require().Len(injector.messages, 1)
	ack := injector.messages[0].GetSensorAck()
	suite.Require().NotNil(ack)
	suite.Equal(central.SensorACK_ACK, ack.GetAction())
	suite.Equal(central.SensorACK_VM_INDEX_REPORT, ack.GetMessageType())
	suite.Equal(vmID, ack.GetResourceId())
	suite.Empty(ack.GetReason())
}

func (suite *PipelineTestSuite) TestRun_NoACKWhenCapabilityMissing() {
	suite.T().Setenv(features.VirtualMachines.EnvVar(), "true")
	vmID := "vm-no-cap"
	msg := createVMIndexMessage(vmID, central.ResourceAction_SYNC_RESOURCE)

	suite.enricher.EXPECT().
		EnrichVirtualMachineWithVulnerabilities(gomock.Any(), gomock.Any()).
		Return(nil)
	suite.vmDatastore.EXPECT().
		UpdateVirtualMachineScan(ctx, vmID, gomock.Any()).
		Return(nil)

	injector := &mockInjectorWithCaps{
		capabilities: map[centralsensor.SensorCapability]bool{},
	}

	err := suite.pipeline.Run(ctx, testClusterID, msg, injector)
	suite.NoError(err)
	suite.Empty(injector.messages, "should not send ACK when SensorACKSupport is missing")
}

func (suite *PipelineTestSuite) TestRun_NoACKOnError() {
	suite.T().Setenv(features.VirtualMachines.EnvVar(), "true")
	vmID := "vm-error"
	msg := createVMIndexMessage(vmID, central.ResourceAction_SYNC_RESOURCE)

	suite.enricher.EXPECT().
		EnrichVirtualMachineWithVulnerabilities(gomock.Any(), gomock.Any()).
		Return(nil)
	suite.vmDatastore.EXPECT().
		UpdateVirtualMachineScan(ctx, vmID, gomock.Any()).
		Return(errors.New("db error"))

	injector := &mockInjectorWithCaps{
		capabilities: map[centralsensor.SensorCapability]bool{
			centralsensor.SensorACKSupport: true,
		},
	}

	err := suite.pipeline.Run(ctx, testClusterID, msg, injector)
	suite.Error(err)
	suite.Empty(injector.messages, "should not send ACK when pipeline errors")
}

func TestSendSensorACK_NACK(t *testing.T) {
	injector := &mockInjectorWithCaps{
		capabilities: map[centralsensor.SensorCapability]bool{
			centralsensor.SensorACKSupport: true,
		},
	}

	common.SendSensorACK(ctx, central.SensorACK_NACK, central.SensorACK_VM_INDEX_REPORT, "vm-nack", "rate limited", injector)

	assert.Len(t, injector.messages, 1)
	ack := injector.messages[0].GetSensorAck()
	assert.NotNil(t, ack)
	assert.Equal(t, central.SensorACK_NACK, ack.GetAction())
	assert.Equal(t, central.SensorACK_VM_INDEX_REPORT, ack.GetMessageType())
	assert.Equal(t, "vm-nack", ack.GetResourceId())
	assert.Equal(t, "rate limited", ack.GetReason())
}

func TestSendSensorACK_NilInjector(t *testing.T) {
	assert.NotPanics(t, func() {
		common.SendSensorACK(ctx, central.SensorACK_ACK, central.SensorACK_VM_INDEX_REPORT, "vm-1", "", nil)
	})
}

func TestSendSensorACK_InjectorWithoutCapabilityCheck(t *testing.T) {
	injector := &mockInjector{}

	common.SendSensorACK(ctx, central.SensorACK_ACK, central.SensorACK_VM_INDEX_REPORT, "vm-1", "", injector)

	assert.Empty(t, injector.messages, "should not send when injector doesn't implement capabilityChecker")
}

func TestPipelineRun_DisabledFeature(t *testing.T) {
	t.Setenv(features.VirtualMachines.EnvVar(), "false")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vmDatastore := vmDatastoreMocks.NewMockDataStore(ctrl)
	enricher := vmEnricherMocks.NewMockVirtualMachineEnricher(ctrl)
	pipeline := &pipelineImpl{
		vmDatastore: vmDatastore,
		enricher:    enricher,
	}

	vmID := "vm-1"
	msg := createVMIndexMessage(vmID, central.ResourceAction_CREATE_RESOURCE)

	err := pipeline.Run(ctx, testClusterID, msg, nil)

	assert.NoError(t, err)
}
