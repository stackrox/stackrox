package index

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/generated/internalapi/central"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/index/mocks"
	vmmetrics "github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"github.com/stretchr/testify/require"
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
	}
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported})
}

func (s *virtualMachineHandlerSuite) TearDownTest() {
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
		err := s.handler.Send(context.Background(), vm, nil)
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
				err := s.handler.Send(ctx, req, nil)
				s.Require().NoError(err)
			}
		}(i)
	}

	// Collect all responses with shorter timeout.
	totalResponses := 0
	for range numGoroutines * numVMsPerGoroutine {
		select {
		case <-s.handler.toCentral:
			totalResponses++
		case <-time.After(500 * time.Millisecond):
			s.T().Logf("Timeout waiting for response, got %d responses", totalResponses)
			return // Don't fail, just exit
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

	// Start draining messages to prevent sendToCentral from blocking
	messageReceived := make(chan bool, 1)
	go func() {
		select {
		case <-s.handler.ResponsesC():
			messageReceived <- true
		case <-time.After(500 * time.Millisecond):
			messageReceived <- false
		}
	}()

	// Test that the goroutine processes sent VMs.
	vm := &v1.IndexReport{VsockCid: cid}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.handler.Send(context.Background(), vm, nil)
		s.Require().NoError(err)
	}()

	wg.Wait()

	// Verify no message was sent
	received := <-messageReceived
	s.Assert().False(received, "Unexpected message to be sent to central")

	// Ensure goroutine has finished processing by draining any remaining messages
	go func() {
		for range s.handler.ResponsesC() {
		}
	}()
	// Wait for goroutine to exit
	select {
	case <-s.handler.stopper.Client().Stopped().Done():
	case <-time.After(time.Second):
	}
}

func (s *virtualMachineHandlerSuite) TestSend_WithDiscoveredFacts_EmitsUpdate() {
	s.T().Setenv(features.VirtualMachines.EnvVar(), "true")
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)
	defer s.handler.Stop()

	vmID := virtualmachine.VMID("vm-update")
	vmInfo := &virtualmachine.Info{
		ID:        vmID,
		Name:      "vm-name",
		Namespace: "vm-namespace",
		Running:   true,
		GuestOS:   "Red Hat Enterprise Linux",
	}
	cid := uint32(1)
	// handleDiscoveredFacts calls GetFromCID, then newMessageToCentral also calls it
	s.store.EXPECT().GetFromCID(cid).Times(2).Return(vmInfo)
	// handleDiscoveredFacts gets previous facts, upserts, then gets again for the update message
	s.store.EXPECT().GetDiscoveredFacts(vmID).Times(1).Return(map[string]string{})
	s.store.EXPECT().UpsertDiscoveredFacts(vmID, gomock.Any()).Times(1)
	s.store.EXPECT().GetDiscoveredFacts(vmID).Times(1).Return(map[string]string{
		virtualmachine.FactsDetectedOSKey: "RHEL",
	})

	discoveredData := &v1.DiscoveredData{
		DetectedOs: v1.DetectedOS_RHEL,
	}

	go func() {
		err := s.handler.Send(context.Background(), &v1.IndexReport{VsockCid: "1"}, discoveredData)
		s.Require().NoError(err)
	}()

	// First message should be the VM update (handleDiscoveredFacts sends it first)
	select {
	case msg := <-s.handler.ResponsesC():
		s.Require().NotNil(msg)
		sensorEvent := msg.GetEvent()
		s.Require().NotNil(sensorEvent)
		s.Assert().Equal(string(vmID), sensorEvent.GetId())
		s.Assert().Equal(central.ResourceAction_UPDATE_RESOURCE, sensorEvent.GetAction())

		vm := sensorEvent.GetVirtualMachine()
		s.Require().NotNil(vm)
		s.Assert().Equal(string(vmID), vm.GetId())
		s.Assert().Equal("Red Hat Enterprise Linux", vm.GetFacts()[virtualmachine.FactsGuestOSKey])
		s.Assert().Equal("RHEL", vm.GetFacts()[virtualmachine.FactsDetectedOSKey])
	case <-time.After(time.Second):
		s.Fail("Expected update message to be sent to central")
	}

	// Second message should be the index report
	select {
	case msg := <-s.handler.ResponsesC():
		s.Require().NotNil(msg)
		sensorEvent := msg.GetEvent()
		s.Require().NotNil(sensorEvent)
		s.Assert().Equal(string(vmID), sensorEvent.GetId())
		s.Assert().Equal(central.ResourceAction_SYNC_RESOURCE, sensorEvent.GetAction())
		s.Assert().NotNil(sensorEvent.GetVirtualMachineIndexReport())
	case <-time.After(time.Second):
		s.Fail("Expected index report message to be sent to central")
	}

	// Drain any remaining messages to prevent goroutine leaks
	go func() {
		for range s.handler.ResponsesC() {
		}
	}()
	time.Sleep(50 * time.Millisecond)
}

func (s *virtualMachineHandlerSuite) TestSend_ErrorPaths() {
	cases := map[string]struct {
		setup   func(t *testing.T, h *handlerImpl, store *mocks.MockVirtualMachineStore) (context.Context, *v1.IndexReport, *v1.DiscoveredData)
		wantErr error
	}{
		"capability missing": {
			setup: func(t *testing.T, h *handlerImpl, store *mocks.MockVirtualMachineStore) (context.Context, *v1.IndexReport, *v1.DiscoveredData) {
				centralcaps.Set(nil)
				return context.Background(), &v1.IndexReport{VsockCid: "1"}, nil
			},
			wantErr: errox.NotImplemented,
		},
		"central not ready": {
			setup: func(t *testing.T, h *handlerImpl, store *mocks.MockVirtualMachineStore) (context.Context, *v1.IndexReport, *v1.DiscoveredData) {
				err := h.Start()
				require.NoError(t, err)
				// Don't notify central as reachable, so centralReady is not done
				return context.Background(), &v1.IndexReport{VsockCid: "1"}, nil
			},
			wantErr: errox.ResourceExhausted,
		},
		"input channel closed": {
			setup: func(t *testing.T, h *handlerImpl, store *mocks.MockVirtualMachineStore) (context.Context, *v1.IndexReport, *v1.DiscoveredData) {
				h.Notify(common.SensorComponentEventCentralReachable)
				return context.Background(), &v1.IndexReport{VsockCid: "1"}, nil
			},
			wantErr: errox.InvariantViolation,
		},
		"context canceled": {
			setup: func(t *testing.T, h *handlerImpl, store *mocks.MockVirtualMachineStore) (context.Context, *v1.IndexReport, *v1.DiscoveredData) {
				h.Notify(common.SensorComponentEventCentralReachable)
				err := h.Start()
				require.NoError(t, err)
				// Set up expectation in case the item gets enqueued before context check
				store.EXPECT().GetFromCID(uint32(1)).Return(&virtualmachine.Info{ID: "vm-1"}).AnyTimes()
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, &v1.IndexReport{VsockCid: "1"}, nil
			},
			wantErr: context.Canceled,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()
			defer centralcaps.Set([]centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported})

			store := mocks.NewMockVirtualMachineStore(ctrl)
			handler := &handlerImpl{
				centralReady: concurrency.NewSignal(),
				lock:         &sync.RWMutex{},
				stopper:      concurrency.NewStopper(),
				store:        store,
			}

			ctx, report, data := tc.setup(s.T(), handler, store)

			// Start draining messages to prevent sendToCentral from blocking (only if handler is started)
			drainDone := make(chan struct{})
			if handler.indexReports != nil {
				go func() {
					defer close(drainDone)
					for {
						select {
						case _, ok := <-handler.ResponsesC():
							if !ok {
								return
							}
						case <-time.After(100 * time.Millisecond):
							// Timeout to prevent blocking forever
							return
						}
					}
				}()
			} else {
				close(drainDone)
			}

			err := handler.Send(ctx, report, data)
			s.Require().Error(err)
			s.ErrorIs(err, tc.wantErr)

			// Clean up: stop handler if it was started
			if handler.indexReports != nil {
				handler.Stop()
				// Wait for goroutine to exit
				select {
				case <-handler.stopper.Client().Stopped().Done():
				case <-time.After(time.Second):
				}
				// Wait for drain goroutine to exit
				select {
				case <-drainDone:
				case <-time.After(200 * time.Millisecond):
				}
			}
		})
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
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.handler.Send(context.Background(), vm, nil)
		s.Require().NoError(err)
	}()

	wg.Wait()

	// Read from ResponsesC to verify message was not sent.
	select {
	case <-s.handler.ResponsesC():
		s.Fail("Unexpected message to be sent to central")
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *virtualMachineHandlerSuite) TestStop() {
	err := s.handler.Start()
	s.Require().NoError(err)

	// Stop should not panic and should stop gracefully.
	s.handler.Stop()

	// Drain any messages that might have been sent
	go func() {
		for range s.handler.ResponsesC() {
		}
	}()

	// Verify stopper is stopped and wait for goroutine to exit
	select {
	case <-s.handler.stopper.Client().Stopped().Done():
		// Expected.
	case <-time.After(time.Second):
		s.Fail("handler should have stopped")
	}
	// Give goroutine time to fully exit
	time.Sleep(50 * time.Millisecond)
}

func (s *virtualMachineHandlerSuite) TestCapabilities() {
	caps := s.handler.Capabilities()
	s.Require().Empty(caps)
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
