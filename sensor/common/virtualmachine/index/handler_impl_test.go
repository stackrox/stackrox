package index

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/index/mocks"
	vmmetrics "github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestVirtualMachineHandler(t *testing.T) {
	suite.Run(t, new(virtualMachineHandlerSuite))
}

type virtualMachineHandlerSuite struct {
	suite.Suite
	ctrl    *gomock.Controller
	store   *mocks.MockVirtualMachineStore
	handler *handlerImpl
}

func (s *virtualMachineHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.store = mocks.NewMockVirtualMachineStore(s.ctrl)
	s.handler = &handlerImpl{
		centralReady: concurrency.NewSignal(),
		lock:         &sync.RWMutex{},
		stopper:      concurrency.NewStopper(),
		store:        s.store,
		pending:      make(map[string]*slot),
	}
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported})
}

func (s *virtualMachineHandlerSuite) TearDownTest() {
	// Reset global capability state to prevent test pollution.
	// If not reset, subsequent tests may incorrectly assume capabilities are present/absent,
	// leading to false negatives that could hide regressions.
	centralcaps.Set(nil)
	goleak.AssertNoGoroutineLeaks(s.T())
}

func (s *virtualMachineHandlerSuite) TestSend() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)
	defer s.handler.Stop()
	s.Require().NotNil(s.handler.toCentral)

	cid := "1"
	s.store.EXPECT().GetFromCID(gomock.Eq(uint32(1))).Times(1).Return(
		&virtualmachine.Info{
			ID: "test-vm",
		})

	// Test that the goroutine processes sent VMs.
	vm := &v1.IndexReport{VsockCid: cid}
	go func() {
		err := s.handler.Send(context.Background(), vm, time.Time{})
		s.Require().NoError(err)
	}()

	// Read from ResponsesC to verify message was sent.
	select {
	case msg := <-s.handler.ResponsesC():
		s.Require().NotNil(msg)
		s.Require().NotNil(msg.MsgFromSensor)

		sensorEvent := msg.GetEvent()
		s.Require().NotNil(sensorEvent)
		s.Assert().Equal("test-vm", sensorEvent.GetId())
		s.Assert().Equal(central.ResourceAction_SYNC_RESOURCE, sensorEvent.GetAction())
		s.Assert().NotNil(sensorEvent.GetVirtualMachineIndexReport())
		s.Assert().Equal("test-vm", sensorEvent.GetVirtualMachineIndexReport().GetId())
	case <-time.After(time.Second):
		s.Fail("Expected message to be sent to central")
	}
}

func (s *virtualMachineHandlerSuite) TestConcurrentSends() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)
	defer s.handler.Stop()

	ctx := context.Background()
	numGoroutines := 3
	numVMsPerGoroutine := 2
	anyOf := func() []any {
		ret := make([]any, 0, numGoroutines*numVMsPerGoroutine)
		cont := 0
		for range numGoroutines {
			for range numVMsPerGoroutine {
				ret = append(ret, uint32(cont))
				cont++
			}
		}
		return ret
	}()
	s.store.EXPECT().GetFromCID(gomock.AnyOf(anyOf...)).Times(numGoroutines * numVMsPerGoroutine).
		Return(
			&virtualmachine.Info{
				ID: "test-vm",
			})

	// Start concurrent sends.
	cont := 0
	mu := sync.Mutex{}
	for i := range numGoroutines {
		go func(routineID int) {
			for range numVMsPerGoroutine {
				var req *v1.IndexReport
				concurrency.WithLock(&mu, func() {
					req = &v1.IndexReport{
						VsockCid: fmt.Sprintf("%d", cont),
					}
					cont++
				})
				err := s.handler.Send(ctx, req, time.Time{})
				s.Require().NoError(err)
			}
		}(i)
	}

	// Collect all responses. The timeout breaks the loop so the Assert
	// below reports the actual vs expected count — do not fail inside
	// the select; that would abort before the diagnostic assertion.
	totalResponses := 0
	for range numGoroutines * numVMsPerGoroutine {
		select {
		case <-s.handler.toCentral:
			totalResponses++
		case <-time.After(500 * time.Millisecond):
			s.T().Logf("Timeout waiting for response, got %d responses", totalResponses)
			s.Assert().Equal(numGoroutines*numVMsPerGoroutine, totalResponses)
			return
		}
	}
	s.Assert().Equal(numGoroutines*numVMsPerGoroutine, totalResponses)
}

func (s *virtualMachineHandlerSuite) TestVirtualMachineNotFound() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)
	defer s.handler.Stop()
	s.Require().NotNil(s.handler.toCentral)

	cid := "1"
	s.store.EXPECT().GetFromCID(gomock.Eq(uint32(1))).Times(1).Return(nil)

	// Test that the goroutine processes sent VMs.
	vm := &v1.IndexReport{VsockCid: cid}
	wg := sync.WaitGroup{}
	wg.Go(func() {
		err := s.handler.Send(context.Background(), vm, time.Time{})
		s.Require().NoError(err)
	})

	wg.Wait()

	// Read from ResponsesC to verify message was not sent.
	select {
	case <-s.handler.ResponsesC():
		s.Fail("Unexpected message to be sent to central")
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *virtualMachineHandlerSuite) TestInvalidCID() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)
	defer s.handler.Stop()
	s.Require().NotNil(s.handler.toCentral)

	cid := "invalid-cid"

	// Test that the goroutine processes sent VMs.
	vm := &v1.IndexReport{VsockCid: cid}
	wg := sync.WaitGroup{}
	wg.Go(func() {
		err := s.handler.Send(context.Background(), vm, time.Time{})
		s.Require().NoError(err)
	})

	wg.Wait()

	// Read from ResponsesC to verify message was not sent.
	select {
	case <-s.handler.ResponsesC():
		s.Fail("Unexpected message to be sent to central")
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *virtualMachineHandlerSuite) TestSend_ResolveByVmID() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)
	defer s.handler.Stop()

	vmID := virtualmachine.VMID("test-vm")
	cases := map[string]struct {
		vsockCID string
	}{
		"should forward when only vm_id is set": {
			vsockCID: "",
		},
		"should prefer vm_id over vsock_cid": {
			vsockCID: "42",
		},
	}
	s.store.EXPECT().Get(gomock.Eq(vmID)).Times(len(cases)).Return(&virtualmachine.Info{ID: vmID})

	for name, tc := range cases {
		s.Run(name, func() {
			go func() {
				err := s.handler.Send(context.Background(), &v1.IndexReport{
					VmId:     string(vmID),
					VsockCid: tc.vsockCID,
				}, time.Time{})
				s.Require().NoError(err)
			}()

			select {
			case msg := <-s.handler.ResponsesC():
				s.Require().NotNil(msg)
				sensorEvent := msg.GetEvent()
				s.Require().NotNil(sensorEvent)
				s.Assert().Equal(string(vmID), sensorEvent.GetId())
				indexEvent := sensorEvent.GetVirtualMachineIndexReport()
				s.Require().NotNil(indexEvent)
				s.Assert().Equal(string(vmID), indexEvent.GetId())
				s.Assert().Equal(string(vmID), indexEvent.GetIndex().GetVmId())
				s.Assert().Equal(tc.vsockCID, indexEvent.GetIndex().GetVsockCid())
			case <-time.After(time.Second):
				s.Fail("Expected message to be sent to central")
			}
		})
	}
}

func (s *virtualMachineHandlerSuite) TestStop() {
	err := s.handler.Start()
	s.Require().NoError(err)

	// Stop should not panic and should stop gracefully.
	s.handler.Stop()

	// Verify stopper is stopped.
	select {
	case <-s.handler.stopper.Client().Stopped().Done():
		// Expected.
	case <-time.After(time.Second):
		s.Fail("handler should have stopped")
	}
}

func (s *virtualMachineHandlerSuite) TestCapabilities() {
	caps := s.handler.Capabilities()
	s.Require().Len(caps, 1)
	s.Contains(caps, centralsensor.SensorACKSupport)
}

func (s *virtualMachineHandlerSuite) TestAccepts() {
	// Should accept SensorACK with VM_INDEX_REPORT type
	vmAckMsg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_SensorAck{SensorAck: &central.SensorACK{
			Action:      central.SensorACK_ACK,
			MessageType: central.SensorACK_VM_INDEX_REPORT,
			ResourceId:  "vm-1",
		}},
	}
	s.Assert().True(s.handler.Accepts(vmAckMsg), "Handler should accept SensorACK for VM_INDEX_REPORT")

	// Should not accept SensorACK with other types
	nodeAckMsg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_SensorAck{SensorAck: &central.SensorACK{
			Action:      central.SensorACK_ACK,
			MessageType: central.SensorACK_NODE_INDEX_REPORT,
			ResourceId:  "node-1",
		}},
	}
	s.Assert().False(s.handler.Accepts(nodeAckMsg), "Handler should not accept SensorACK for NODE_INDEX_REPORT")

	// Should not accept other message types
	otherMsg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ClusterConfig{},
	}
	s.Assert().False(s.handler.Accepts(otherMsg), "Handler should not accept other message types")
}

func (s *virtualMachineHandlerSuite) TestProcessMessage() {
	ctx := context.Background()

	getMetric := func(label string) float64 {
		return testutil.ToFloat64(vmmetrics.IndexReportAcksReceived.WithLabelValues(label))
	}

	cases := map[string]struct {
		msg        *central.MsgToSensor
		expectAck  int
		expectNack int
	}{
		"ack increments ack metric": {
			msg: &central.MsgToSensor{
				Msg: &central.MsgToSensor_SensorAck{SensorAck: &central.SensorACK{
					Action:      central.SensorACK_ACK,
					MessageType: central.SensorACK_VM_INDEX_REPORT,
					ResourceId:  "vm-ack",
				}},
			},
			expectAck:  1,
			expectNack: 0,
		},
		"nack increments nack metric": {
			msg: &central.MsgToSensor{
				Msg: &central.MsgToSensor_SensorAck{SensorAck: &central.SensorACK{
					Action:      central.SensorACK_NACK,
					MessageType: central.SensorACK_VM_INDEX_REPORT,
					ResourceId:  "vm-nack",
					Reason:      "rate limited",
				}},
			},
			expectAck:  0,
			expectNack: 1,
		},
		"non-VM message does not change metrics": {
			msg: &central.MsgToSensor{
				Msg: &central.MsgToSensor_SensorAck{SensorAck: &central.SensorACK{
					Action:      central.SensorACK_ACK,
					MessageType: central.SensorACK_NODE_INDEX_REPORT,
					ResourceId:  "node-1",
				}},
			},
			expectAck:  0,
			expectNack: 0,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			initialAck := getMetric(central.SensorACK_ACK.String())
			initialNack := getMetric(central.SensorACK_NACK.String())

			err := s.handler.ProcessMessage(ctx, tc.msg)
			s.Require().NoError(err)
			s.Equal(initialAck+float64(tc.expectAck), getMetric(central.SensorACK_ACK.String()))
			s.Equal(initialNack+float64(tc.expectNack), getMetric(central.SensorACK_NACK.String()))
		})
	}
}

func (s *virtualMachineHandlerSuite) TestResponsesC_BeforeStart() {
	s.Assert().Panics(func() { _ = s.handler.ResponsesC() })
}

func (s *virtualMachineHandlerSuite) TestResponsesC_AfterStart() {
	err := s.handler.Start()
	s.Require().NoError(err)
	defer s.handler.Stop()

	ch := s.handler.ResponsesC()
	s.Require().NotNil(ch)
}

func (s *virtualMachineHandlerSuite) TestComplianceC_AfterStart() {
	err := s.handler.Start()
	s.Require().NoError(err)
	defer s.handler.Stop()

	ch := s.handler.ComplianceC()
	s.Require().NotNil(ch, "ComplianceC channel should be not nil after start")
}

func (s *virtualMachineHandlerSuite) TestForwardToCompliance() {
	err := s.handler.Start()
	s.Require().NoError(err)
	defer s.handler.Stop()

	cases := map[string]struct {
		centralAction  central.SensorACK_Action
		resourceID     string
		vmID           string
		nodeName       string
		reason         string
		expectedAction sensor.MsgToCompliance_ComplianceACK_Action
	}{
		"should forward ACK to compliance with composite resource ID": {
			centralAction:  central.SensorACK_ACK,
			resourceID:     "vm-123:100",
			vmID:           "vm-123",
			nodeName:       "node-a",
			reason:         "all good",
			expectedAction: sensor.MsgToCompliance_ComplianceACK_ACK,
		},
		"should forward NACK to compliance with composite resource ID": {
			centralAction:  central.SensorACK_NACK,
			resourceID:     "vm-456:200",
			vmID:           "vm-456",
			nodeName:       "node-b",
			reason:         "validation failed",
			expectedAction: sensor.MsgToCompliance_ComplianceACK_NACK,
		},
		"should forward ACK with bare VM ID": {
			centralAction:  central.SensorACK_ACK,
			resourceID:     "vm-789",
			vmID:           "vm-789",
			nodeName:       "node-c",
			reason:         "",
			expectedAction: sensor.MsgToCompliance_ComplianceACK_ACK,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			s.store.EXPECT().Get(virtualmachine.VMID(tc.vmID)).Return(
				&virtualmachine.Info{ID: virtualmachine.VMID(tc.vmID), NodeName: tc.nodeName})

			ctx := context.Background()
			msg := &central.MsgToSensor{
				Msg: &central.MsgToSensor_SensorAck{SensorAck: &central.SensorACK{
					Action:      tc.centralAction,
					MessageType: central.SensorACK_VM_INDEX_REPORT,
					ResourceId:  tc.resourceID,
					Reason:      tc.reason,
				}},
			}

			err := s.handler.ProcessMessage(ctx, msg)
			s.Require().NoError(err)

			select {
			case got := <-s.handler.ComplianceC():
				ack := got.Msg.GetComplianceAck()
				s.Require().NotNil(ack)
				s.Equal(tc.expectedAction, ack.GetAction())
				s.Equal(sensor.MsgToCompliance_ComplianceACK_VM_INDEX_REPORT, ack.GetMessageType())
				s.Equal(tc.resourceID, ack.GetResourceId())
				s.Equal(tc.reason, ack.GetReason())
				s.Equal(tc.nodeName, got.Hostname)
				s.False(got.Broadcast)
			case <-time.After(100 * time.Millisecond):
				s.Fail("timed out waiting for compliance message")
			}
		})
	}
}

func (s *virtualMachineHandlerSuite) TestForwardToCompliance_DropsWhenVMUnknown() {
	err := s.handler.Start()
	s.Require().NoError(err)
	defer s.handler.Stop()

	cases := map[string]struct {
		resourceID string
		expectedID string
	}{
		"should drop when resource ID is empty": {
			resourceID: "",
		},
		"should drop when VM not in store": {
			resourceID: "vm-deleted",
			expectedID: "vm-deleted",
		},
		"should drop when VM from composite ID not in store": {
			resourceID: "vm-deleted:999",
			expectedID: "vm-deleted",
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			if tc.expectedID != "" {
				s.store.EXPECT().Get(virtualmachine.VMID(tc.expectedID)).Return(nil)
			}

			ctx := context.Background()
			msg := &central.MsgToSensor{
				Msg: &central.MsgToSensor_SensorAck{SensorAck: &central.SensorACK{
					Action:      central.SensorACK_ACK,
					MessageType: central.SensorACK_VM_INDEX_REPORT,
					ResourceId:  tc.resourceID,
				}},
			}

			err := s.handler.ProcessMessage(ctx, msg)
			s.Require().NoError(err)

			select {
			case <-s.handler.ComplianceC():
				s.Fail("ACK should have been dropped when VM is unknown, not forwarded")
			default:
			}
		})
	}
}

func (s *virtualMachineHandlerSuite) TestForwardToCompliance_NoStartDoesNotPanic() {
	ctx := context.Background()
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_SensorAck{SensorAck: &central.SensorACK{
			Action:      central.SensorACK_ACK,
			MessageType: central.SensorACK_VM_INDEX_REPORT,
			ResourceId:  "vm-no-start",
		}},
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = s.handler.ProcessMessage(ctx, msg)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		s.Fail("ProcessMessage should not block when Start() has not been called")
	}

	s.Nil(s.handler.ComplianceC())
}

func (s *virtualMachineHandlerSuite) TestForwardToCompliance_UnknownActionDropped() {
	err := s.handler.Start()
	s.Require().NoError(err)
	defer s.handler.Stop()

	ctx := context.Background()
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_SensorAck{SensorAck: &central.SensorACK{
			Action:      central.SensorACK_Action(999),
			MessageType: central.SensorACK_VM_INDEX_REPORT,
			ResourceId:  "vm-unknown",
		}},
	}

	err = s.handler.ProcessMessage(ctx, msg)
	s.Require().NoError(err)

	select {
	case <-s.handler.ComplianceC():
		s.Fail("unexpected message on ComplianceC for unknown SensorACK action")
	default:
	}
}

func (s *virtualMachineHandlerSuite) TestForwardToCompliance_DoesNotBlockWhenStopped() {
	err := s.handler.Start()
	s.Require().NoError(err)

	// Fill the compliance channel to capacity.
	s.handler.toCompliance <- common.MessageToComplianceWithAddress{}

	s.handler.Stop()

	s.store.EXPECT().Get(virtualmachine.VMID("vm-stopped")).Return(
		&virtualmachine.Info{ID: "vm-stopped", NodeName: "node-x"})

	ctx := context.Background()
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_SensorAck{SensorAck: &central.SensorACK{
			Action:      central.SensorACK_NACK,
			MessageType: central.SensorACK_VM_INDEX_REPORT,
			ResourceId:  "vm-stopped",
			Reason:      "after stop",
		}},
	}

	// The first select in forwardToCompliance sees StopRequested and
	// returns before attempting the channel send.
	err = s.handler.ProcessMessage(ctx, msg)
	s.Require().NoError(err)

	// Only the pre-filled message should be in the channel.
	select {
	case <-s.handler.ComplianceC():
	default:
		s.Fail("expected the pre-filled message")
	}
	select {
	case <-s.handler.ComplianceC():
		s.Fail("the second message should have been dropped")
	default:
	}
}

func (s *virtualMachineHandlerSuite) TestForwardToCompliance_DropsWhenQueueFull() {
	err := s.handler.Start()
	s.Require().NoError(err)
	defer s.handler.Stop()

	s.store.EXPECT().Get(virtualmachine.VMID("vm-first")).Return(
		&virtualmachine.Info{ID: "vm-first", NodeName: "node-1"})
	s.store.EXPECT().Get(virtualmachine.VMID("vm-second")).Return(
		&virtualmachine.Info{ID: "vm-second", NodeName: "node-2"})

	ctx := context.Background()
	makeMsg := func(id string) *central.MsgToSensor {
		return &central.MsgToSensor{
			Msg: &central.MsgToSensor_SensorAck{SensorAck: &central.SensorACK{
				Action:      central.SensorACK_ACK,
				MessageType: central.SensorACK_VM_INDEX_REPORT,
				ResourceId:  id,
			}},
		}
	}

	// First send fills the buffer (capacity 1).
	err = s.handler.ProcessMessage(ctx, makeMsg("vm-first"))
	s.Require().NoError(err)

	// Second send without draining: hits the default branch, drops
	// immediately instead of blocking. This protects ProcessMessage
	// from stalling if compliance is slow.
	err = s.handler.ProcessMessage(ctx, makeMsg("vm-second"))
	s.Require().NoError(err)

	// Only the first message is in the channel.
	select {
	case got := <-s.handler.ComplianceC():
		s.Equal("vm-first", got.Msg.GetComplianceAck().GetResourceId())
	case <-time.After(100 * time.Millisecond):
		s.Fail("first message should be available in the buffer")
	}

	// Channel should now be empty (second was dropped).
	select {
	case <-s.handler.ComplianceC():
		s.Fail("second message should have been dropped when queue was full")
	default:
	}
}

func (s *virtualMachineHandlerSuite) TestForwardToCompliance_DropsOnCancelledContext() {
	err := s.handler.Start()
	s.Require().NoError(err)
	defer s.handler.Stop()

	// Fill the buffer so the send would need to block.
	s.handler.toCompliance <- common.MessageToComplianceWithAddress{}

	s.store.EXPECT().Get(virtualmachine.VMID("vm-cancelled")).Return(
		&virtualmachine.Info{ID: "vm-cancelled", NodeName: "node-c"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_SensorAck{SensorAck: &central.SensorACK{
			Action:      central.SensorACK_ACK,
			MessageType: central.SensorACK_VM_INDEX_REPORT,
			ResourceId:  "vm-cancelled",
		}},
	}

	// The first select in forwardToCompliance sees ctx.Done() and
	// returns before attempting the channel send.
	err = s.handler.ProcessMessage(ctx, msg)
	s.Require().NoError(err)

	// Drain the pre-filled message; the cancelled one was dropped.
	select {
	case <-s.handler.ComplianceC():
	default:
		s.Fail("expected the pre-filled message")
	}
	select {
	case <-s.handler.ComplianceC():
		s.Fail("the message with cancelled context should have been dropped")
	default:
	}
}

func (s *virtualMachineHandlerSuite) TestSend_CapabilityNotSupported() {
	// Remove capability to simulate old Central version
	centralcaps.Set(nil)

	cid := "1"
	vm := &v1.IndexReport{VsockCid: cid}

	// Send should return errCapabilityNotSupported when capability is absent
	err := s.handler.Send(context.Background(), vm, time.Time{})
	s.Require().Error(err)
	s.Require().ErrorIs(err, errCapabilityNotSupported)
	s.Require().ErrorIs(err, errox.NotImplemented)
	s.Assert().ErrorContains(err, "Central does not have virtual machine capability")
}

func (s *virtualMachineHandlerSuite) TestSend_AfterStop() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Stop()

	vm := &v1.IndexReport{VsockCid: "1"}
	err = s.handler.Send(context.Background(), vm, time.Time{})
	s.Require().Error(err)
	s.Require().ErrorIs(err, errInputChanClosed)
	s.Require().ErrorIs(err, errox.InvariantViolation)
}

func (s *virtualMachineHandlerSuite) TestSend_CentralNotReachable() {
	err := s.handler.Start()
	s.Require().NoError(err)
	defer s.handler.Stop()

	vm := &v1.IndexReport{VsockCid: "1"}
	err = s.handler.Send(context.Background(), vm, time.Time{})
	s.Require().Error(err)
	s.Require().ErrorIs(err, errCentralNotReachable)
	s.Require().ErrorIs(err, errox.ResourceExhausted)
}

func (s *virtualMachineHandlerSuite) TestStart_CalledTwice() {
	err := s.handler.Start()
	s.Require().NoError(err)
	defer s.handler.Stop()

	err = s.handler.Start()
	s.Require().ErrorIs(err, errStartMoreThanOnce)
}

// reactiveLatencySampleCount reads the current sample count (number of
// Observe() calls) of the SLA-measurement histogram directly off the
// collector, since it's a plain Histogram (not a vec) shared across tests:
// testutil.CollectAndCount would only ever report 1 (one time series exists),
// not how many observations were made, so callers must compare deltas.
func (s *virtualMachineHandlerSuite) reactiveLatencySampleCount() uint64 {
	var m dto.Metric
	s.Require().NoError(vmmetrics.VirtualMachineReactiveIndexReportLatencySeconds.Write(&m))
	return m.GetHistogram().GetSampleCount()
}

func (s *virtualMachineHandlerSuite) TestSend_FairFIFO_ReactiveDoesNotJumpAheadOfOtherVMs() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)
	defer s.handler.Stop()

	report1 := &v1.IndexReport{VsockCid: "1"}
	report2 := &v1.IndexReport{VsockCid: "2"}
	report3 := &v1.IndexReport{VsockCid: "3"}

	s.store.EXPECT().GetFromCID(uint32(1)).Return(&virtualmachine.Info{ID: "vm-1"}).AnyTimes()
	s.store.EXPECT().GetFromCID(uint32(2)).Return(&virtualmachine.Info{ID: "vm-2"}).AnyTimes()
	s.store.EXPECT().GetFromCID(uint32(3)).Return(&virtualmachine.Info{ID: "vm-3"}).AnyTimes()

	initialLatencyCount := s.reactiveLatencySampleCount()

	// report1 is picked up by run()'s single goroutine and blocks trying to
	// hand it to ResponsesC() (nobody is reading yet), which pins the loop
	// mid-iteration before it can look at anything queued afterward.
	s.Require().NoError(s.handler.Send(context.Background(), report1, time.Time{}))

	// While run() is blocked delivering report1, queue a scheduled backlog
	// entry for a different VM (report2), then a reactive entry for yet
	// another VM (report3). Fair FIFO means report3 must NOT come out ahead
	// of the already-queued report2, even though it's reactive.
	// Give run() a moment to actually reach the blocking send for report1
	// before queueing more — this is a small, generous sleep, not a tight
	// race: the assertions below tolerate the goroutine being slightly
	// slower, they just require it to not have consumed report2 yet, which
	// it structurally cannot do until report1's send unblocks.
	time.Sleep(50 * time.Millisecond)
	s.Require().NoError(s.handler.Send(context.Background(), report2, time.Time{}))
	s.Require().NoError(s.handler.Send(context.Background(), report3, time.Now()))

	var received []string
	for range 3 {
		select {
		case msg := <-s.handler.ResponsesC():
			received = append(received, msg.GetEvent().GetId())
		case <-time.After(time.Second):
			s.Fail("timed out waiting for a message", "received so far: %v", received)
		}
	}

	s.Require().Len(received, 3)
	s.Equal("vm-1", received[0], "report1 was already in flight before report3 existed")
	s.Equal("vm-2", received[1], "report2 was queued first among the two backlog entries")
	s.Equal("vm-3", received[2], "a reactive report must not jump ahead of an already-queued different VM")

	// Only report3 (the reactive one, non-zero generatedAt) should have
	// observed the SLA latency histogram.
	s.Equal(initialLatencyCount+1, s.reactiveLatencySampleCount(),
		"exactly one observation (the reactive report) should be recorded; the two scheduled reports must not add samples")
}

func (s *virtualMachineHandlerSuite) TestSend_CoalescesSecondSendForSameVMKeepingQueuePosition() {
	// Deliberately does not call Start(): no run() consumer goroutine means
	// pending/indexReports are only ever touched by this test goroutine, so
	// they can be inspected directly and deterministically.
	s.handler.indexReports = make(chan *slot, 2)
	s.handler.centralReady.Signal()

	older := &v1.IndexReport{VsockCid: "9"}
	other := &v1.IndexReport{VsockCid: "10"}
	newer := &v1.IndexReport{VsockCid: "9"}

	s.Require().NoError(s.handler.Send(context.Background(), older, time.Time{}))
	s.Require().NoError(s.handler.Send(context.Background(), other, time.Time{}))
	generatedAt := time.Now()
	s.Require().NoError(s.handler.Send(context.Background(), newer, generatedAt))

	s.Require().Len(s.handler.pending, 2, "coalescing must not add a second map entry for vsock_cid=9")

	// Exactly two slots ever reached the channel: "older"/"newer" coalesced
	// into one slot at "older"'s original (first) queue position, and
	// "other" is a distinct, second slot — never a third entry.
	first := <-s.handler.indexReports
	s.Same(newer, first.report, "the coalesced slot must carry the latest report")
	s.Equal(generatedAt, first.generatedAt)
	s.Equal("9", first.vsockCID)

	second := <-s.handler.indexReports
	s.Same(other, second.report, "the other VM's slot must be unaffected and still second in line")
}

func (s *virtualMachineHandlerSuite) TestSend_QueuesFreshSlotOnceInFlightSlotIsConsumed() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)
	defer s.handler.Stop()

	s.store.EXPECT().GetFromCID(uint32(5)).Return(&virtualmachine.Info{ID: "vm-5"}).AnyTimes()

	first := &v1.IndexReport{VsockCid: "5"}
	s.Require().NoError(s.handler.Send(context.Background(), first, time.Time{}))

	select {
	case msg := <-s.handler.ResponsesC():
		s.Equal("vm-5", msg.GetEvent().GetId())
	case <-time.After(time.Second):
		s.Fail("timed out waiting for the first message")
	}

	// run() clears the pending entry before handing the slot's contents to
	// handleIndexReport, but that happens on its own goroutine; poll briefly
	// rather than asserting immediately after the receive above.
	s.Eventually(func() bool {
		s.handler.slotsMu.Lock()
		defer s.handler.slotsMu.Unlock()
		return len(s.handler.pending) == 0
	}, time.Second, 10*time.Millisecond, "expected vm-5's pending entry to be cleared after consumption")

	second := &v1.IndexReport{VsockCid: "5"}
	s.Require().NoError(s.handler.Send(context.Background(), second, time.Time{}))

	select {
	case msg := <-s.handler.ResponsesC():
		s.Equal("vm-5", msg.GetEvent().GetId())
	case <-time.After(time.Second):
		s.Fail("timed out waiting for the second message")
	}
}

func (s *virtualMachineHandlerSuite) TestSend_ConcurrentSendsCoalesceRaceFreeAndBounded() {
	// Deliberately does not call Start(): no run() consumer goroutine
	// draining indexReports means the map's final contents can be
	// inspected deterministically once all senders finish, while -race
	// still exercises real concurrent access to slotsMu/pending from every
	// sending goroutine below.
	const numVMs = 40
	const sendsPerVM = 25

	s.handler.indexReports = make(chan *slot, numVMs)
	s.handler.centralReady.Signal()

	// Repeated concurrent sends per VM exercise the coalescing path under
	// contention, not just the first-insert path.
	wg := sync.WaitGroup{}
	for i := range numVMs {
		cid := fmt.Sprintf("%d", i)
		for range sendsPerVM {
			wg.Go(func() {
				report := &v1.IndexReport{VsockCid: cid}
				err := s.handler.Send(context.Background(), report, time.Now())
				s.Require().NoError(err)
			})
		}
	}
	wg.Wait()

	// Exactly one slot per VM ever reached the channel, regardless of how
	// many sends raced for the same VM: indexReports must never hold more
	// than one entry per VM at a time.
	s.Len(s.handler.indexReports, numVMs)
	seen := set.NewStringSet()
	for range numVMs {
		got := <-s.handler.indexReports
		s.True(seen.Add(got.vsockCID), "each VM must appear at most once in indexReports, got a duplicate for %s", got.vsockCID)
	}
}

func (s *virtualMachineHandlerSuite) TestSend_ConcurrentWithStopDoesNotPanicOnNilPendingMap() {
	// Regression test: checkSendPreconditions only checks h.stopper, not
	// h.pending, so a Send() can pass that check and then race Stop(). Stop()
	// must never leave a window where such a Send() writes to a nilled-out
	// pending map (see the comment on h.pending in Stop()). Success here is
	// simply "no panic, no goroutine leak" under -race; drive many
	// concurrent senders against a single Stop() to make the race window
	// likely to be hit.
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)

	s.store.EXPECT().GetFromCID(gomock.Any()).Return(&virtualmachine.Info{ID: "vm"}).AnyTimes()

	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for range s.handler.ResponsesC() {
		}
	}()

	const numSenders = 50
	wg := sync.WaitGroup{}
	for i := range numSenders {
		cid := fmt.Sprintf("%d", i)
		wg.Go(func() {
			// Mirrors real callers (e.g. service_impl.go), which always pass
			// a bounded context: enqueueScheduled blocks on the channel send
			// with no deadline of its own, so without a timeout here a Send()
			// racing Stop() after the run() consumer has exited would block
			// forever instead of returning an error.
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			report := &v1.IndexReport{VsockCid: cid}
			// The error is intentionally ignored: Send legitimately fails
			// once Stop() wins the race against it. The only property this
			// test asserts is that it never panics.
			_ = s.handler.Send(ctx, report, time.Time{})
		})
	}

	s.handler.Stop()
	wg.Wait()
	<-drainDone
}

func TestVmIDFromResourceID(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		resourceID string
		expected   string
	}{
		"extracts vm id from composite resource id": {
			resourceID: "vm-1:100",
			expected:   "vm-1",
		},
		"returns bare vm id as-is": {
			resourceID: "vm-1",
			expected:   "vm-1",
		},
		"returns empty string for empty input": {
			resourceID: "",
			expected:   "",
		},
		"extracts uuid vm id from composite": {
			resourceID: "d2696fad-8ef2-49f5-9726-499b1419be20:1289650420",
			expected:   "d2696fad-8ef2-49f5-9726-499b1419be20",
		},
		"returns bare CID as-is": {
			resourceID: "1289650420",
			expected:   "1289650420",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, vmIDFromResourceID(tc.resourceID))
		})
	}
}
