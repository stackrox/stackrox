package index

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/index/mocks"
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

	cid := uint32(1)
	s.store.EXPECT().Get(gomock.Eq(virtualmachine.VMID("test-vm"))).Times(1).Return(
		&virtualmachine.Info{
			ID: "test-vm",
		})

	go func() {
		err := s.handler.Send(context.Background(), &virtualmachine.Info{
			ID:       "test-vm",
			VSOCKCID: &cid,
		}, &v4.IndexReport{State: "IndexFinished"})
		s.Require().NoError(err)
	}()

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
		index := sensorEvent.GetVirtualMachineIndexReport().GetIndex()
		s.Require().NotNil(index)
		s.Assert().Equal("test-vm", index.GetVmId())
		s.Assert().Equal("1", index.GetVsockCid())
		s.Assert().Equal("IndexFinished", index.GetIndexV4().GetState())
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
	s.store.EXPECT().Get(gomock.Any()).Times(numGoroutines * numVMsPerGoroutine).
		DoAndReturn(func(id virtualmachine.VMID) *virtualmachine.Info {
			return &virtualmachine.Info{ID: id}
		})

	cont := 0
	mu := sync.Mutex{}
	for range numGoroutines {
		go func() {
			for range numVMsPerGoroutine {
				var cid uint32
				concurrency.WithLock(&mu, func() {
					cid = uint32(cont)
					cont++
				})
				err := s.handler.Send(ctx, &virtualmachine.Info{
					ID:       virtualmachine.VMID("test-vm"),
					VSOCKCID: &cid,
				}, &v4.IndexReport{})
				s.Require().NoError(err)
			}
		}()
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

	cid := uint32(1)
	s.store.EXPECT().GetFromCID(gomock.Eq(uint32(1))).Times(1).Return(nil)

	wg := sync.WaitGroup{}
	wg.Go(func() {
		// Empty ID forces CID-based resolution in the processing path.
		err := s.handler.Send(context.Background(), &virtualmachine.Info{
			VSOCKCID: &cid,
		}, &v4.IndexReport{})
		s.Require().NoError(err)
	})

	wg.Wait()

	// No message should be forwarded when the VM cannot be resolved.
	select {
	case <-s.handler.ResponsesC():
		s.Fail("Unexpected message to be sent to central")
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *virtualMachineHandlerSuite) TestResolveVM_InvalidCID() {
	_, _, err := s.handler.resolveVM(&v1.IndexReport{VsockCid: "invalid-cid"})
	s.Require().Error(err)
	s.Assert().ErrorContains(err, "invalid Vsock CID")
}

func (s *virtualMachineHandlerSuite) TestSend_ResolveByVmID() {
	vmID := virtualmachine.VMID("test-vm")
	cases := map[string]struct {
		vsockCID *uint32
		wantCID  string
	}{
		"should forward when only vm_id is set": {
			vsockCID: nil,
			wantCID:  "",
		},
		"should include vsock_cid when known": {
			vsockCID: new(uint32(42)),
			wantCID:  "42",
		},
	}
	s.store.EXPECT().Get(gomock.Eq(vmID)).Times(len(cases)).Return(&virtualmachine.Info{ID: vmID})

	for name, tc := range cases {
		s.Run(name, func() {
			synctest.Test(s.T(), func(t *testing.T) {
				handler := &handlerImpl{
					centralReady: concurrency.NewSignal(),
					lock:         &sync.RWMutex{},
					stopper:      concurrency.NewStopper(),
					store:        s.store,
				}
				err := handler.Start()
				s.Require().NoError(err)
				handler.Notify(common.SensorComponentEventCentralReachable)
				defer handler.Stop()

				go func() {
					err := handler.Send(context.Background(), &virtualmachine.Info{
						ID:       vmID,
						VSOCKCID: tc.vsockCID,
					}, &v4.IndexReport{})
					s.Require().NoError(err)
				}()

				synctest.Wait()

				msg := <-handler.ResponsesC()
				s.Require().NotNil(msg)
				sensorEvent := msg.GetEvent()
				s.Require().NotNil(sensorEvent)
				s.Assert().Equal(string(vmID), sensorEvent.GetId())
				indexEvent := sensorEvent.GetVirtualMachineIndexReport()
				s.Require().NotNil(indexEvent)
				s.Assert().Equal(string(vmID), indexEvent.GetId())
				s.Assert().Equal(string(vmID), indexEvent.GetIndex().GetVmId())
				s.Assert().Equal(tc.wantCID, indexEvent.GetIndex().GetVsockCid())
			})
		})
	}
}

func (s *virtualMachineHandlerSuite) TestSend_NilVM() {
	err := s.handler.Send(context.Background(), nil, &v4.IndexReport{})
	s.Require().Error(err)
	s.Assert().ErrorContains(err, "virtual machine info is nil")
}

func (s *virtualMachineHandlerSuite) TestStop() {
	err := s.handler.Start()
	s.Require().NoError(err)

	s.handler.Stop()

	select {
	case <-s.handler.stopper.Client().Stopped().Done():
	case <-time.After(time.Second):
		s.Fail("handler should have stopped")
	}
}

func (s *virtualMachineHandlerSuite) TestCapabilities() {
	caps := s.handler.Capabilities()
	s.Require().Len(caps, 1)
	s.Contains(caps, centralsensor.SensorACKSupport)
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

func (s *virtualMachineHandlerSuite) TestSend_CapabilityNotSupported() {
	// Simulate an older Central that lacks VirtualMachinesSupported.
	centralcaps.Set(nil)

	err := s.handler.Send(context.Background(), &virtualmachine.Info{ID: "vm-1"}, &v4.IndexReport{})
	s.Require().Error(err)
	s.Require().ErrorIs(err, errCapabilityNotSupported)
	s.Require().ErrorIs(err, errox.NotImplemented)
	s.Assert().ErrorContains(err, "Central does not have virtual machine capability")
}

func (s *virtualMachineHandlerSuite) TestSend_AfterStop() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Stop()

	err = s.handler.Send(context.Background(), &virtualmachine.Info{ID: "vm-1"}, &v4.IndexReport{})
	s.Require().Error(err)
	s.Require().ErrorIs(err, errInputChanClosed)
	s.Require().ErrorIs(err, errox.InvariantViolation)
}

func (s *virtualMachineHandlerSuite) TestSend_CentralNotReachable() {
	err := s.handler.Start()
	s.Require().NoError(err)
	defer s.handler.Stop()

	err = s.handler.Send(context.Background(), &virtualmachine.Info{ID: "vm-1"}, &v4.IndexReport{})
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
