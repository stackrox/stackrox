package sensor

import (
	"context"
	"io"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	mocksClient "github.com/stackrox/rox/sensor/common/sensor/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type centralReceiverSuite struct {
	suite.Suite
	controller *gomock.Controller
	mockClient *mocksClient.MockServiceCommunicateClient
	receiver   CentralReceiver
	finished   *sync.WaitGroup
}

var _ suite.SetupTestSuite = (*centralReceiverSuite)(nil)

func (s *centralReceiverSuite) SetupTest() {
	s.controller = gomock.NewController(s.T())
	s.mockClient = mocksClient.NewMockServiceCommunicateClient(s.controller)
	s.finished = &sync.WaitGroup{}
	s.finished.Add(1)
}

func (s *centralReceiverSuite) TearDownTest() {
	if s.receiver != nil {
		s.receiver.Stop()
		s.finished.Wait()
	}
	goleak.AssertNoGoroutineLeaks(s.T())
}

func Test_CentralReceiverSuite(t *testing.T) {
	suite.Run(t, new(centralReceiverSuite))
}

func (s *centralReceiverSuite) Test_StreamContextCancelShouldStopFlow() {
	s.receiver = NewCentralReceiver(s.finished)
	streamContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.mockClient.EXPECT().Context().AnyTimes().Return(streamContext).AnyTimes()

	s.mockClient.EXPECT().Recv().Times(1).DoAndReturn(func() (*central.MsgToSensor, error) {
		testMsg := &central.MsgToSensor{
			Msg: &central.MsgToSensor_PolicySync{
				PolicySync: &central.PolicySync{
					Policies: []*storage.Policy{},
				},
			},
		}
		cancel()
		s.T().Logf("Canceled context")
		return testMsg, nil
	})

	s.receiver.Start(s.mockClient)
	s.finished.Wait()
	s.EqualError(s.receiver.Stopped().Err(), "context canceled")
}

func (s *centralReceiverSuite) Test_RecvErrorShouldStopFlow() {
	s.receiver = NewCentralReceiver(s.finished)
	s.mockClient.EXPECT().Context().AnyTimes().Return(context.Background()).AnyTimes()

	s.mockClient.EXPECT().Recv().Times(1).DoAndReturn(func() (*central.MsgToSensor, error) {
		return nil, errors.New("some error")
	})

	s.receiver.Start(s.mockClient)
	s.finished.Wait()
	s.EqualError(s.receiver.Stopped().Err(), "some error")
}

func (s *centralReceiverSuite) Test_EofShouldStopFlowWithNoError() {
	s.receiver = NewCentralReceiver(s.finished)
	s.mockClient.EXPECT().Context().AnyTimes().Return(context.Background()).AnyTimes()

	s.mockClient.EXPECT().Recv().Times(1).DoAndReturn(func() (*central.MsgToSensor, error) {
		return nil, io.EOF
	})

	s.receiver.Start(s.mockClient)
	s.finished.Wait()
	s.NoError(s.receiver.Stopped().Err())
}

func (s *centralReceiverSuite) Test_SlowComponentDoesNotBlockOthers() {
	// Let's create 2 components. Fast that will process messages as they appear.
	// And blocking that will be blocked and never process any message.
	fastTick := make(chan struct{})
	close(fastTick)
	fastComponent := &testSensorComponent{
		name:       "fast",
		tick:       fastTick,
		responsesC: make(chan *message.ExpiringMessage),
	}
	blockedChan := make(chan struct{})
	blockingComponent := &testSensorComponent{
		name:       "blocked",
		tick:       blockedChan,
		responsesC: make(chan *message.ExpiringMessage),
	}

	components := []common.SensorComponent{fastComponent, blockingComponent}
	s.receiver = NewCentralReceiver(s.finished, components...)

	s.mockClient.EXPECT().Context().AnyTimes().Return(context.Background()).AnyTimes()

	msgCount := 3
	s.mockClient.EXPECT().Recv().MinTimes(msgCount).DoAndReturn(func() (*central.MsgToSensor, error) {
		testMsg := &central.MsgToSensor{
			Msg: &central.MsgToSensor_PolicySync{
				PolicySync: &central.PolicySync{
					Policies: []*storage.Policy{},
				},
			},
		}
		return testMsg, nil
	})

	s.receiver.Start(s.mockClient)

	for i := 0; i < msgCount; i++ {
		select {
		case <-fastComponent.ResponsesC():
			s.T().Logf("Fast component processed the message")
		case <-blockingComponent.ResponsesC():
			assert.FailNow(s.T(), "blocked component should not process the message")
		}
	}

	s.receiver.Stop()
	s.finished.Wait()
	s.NoError(s.receiver.Stopped().Err())
}

type testSensorComponent struct {
	name       string
	tick       <-chan struct{}
	responsesC chan *message.ExpiringMessage
}

func (t *testSensorComponent) ProcessMessage(ctx context.Context, _ *central.MsgToSensor) error {
	select {
	case <-t.tick:
		select {
		case t.responsesC <- &message.ExpiringMessage{}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *testSensorComponent) Name() string {
	return t.name
}

func (t *testSensorComponent) Notify(common.SensorComponentEvent) {}

func (t *testSensorComponent) Start() error {
	return nil
}

func (t *testSensorComponent) Stop() {
	close(t.responsesC)
}

func (t *testSensorComponent) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{}
}

func (t *testSensorComponent) ResponsesC() <-chan *message.ExpiringMessage {
	return t.responsesC
}

var _ common.SensorComponent = (*testSensorComponent)(nil)
