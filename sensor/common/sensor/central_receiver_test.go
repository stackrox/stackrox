package sensor

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/metrics"
	mocksClient "github.com/stackrox/rox/sensor/common/sensor/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	totalMetric  = "rox_sensor_component_queue_operations_total"
	errorsMetric = "rox_sensor_component_process_message_errors_total"
)

type centralReceiverSuite struct {
	suite.Suite
	controller *gomock.Controller
	mockClient *mocksClient.MockServiceCommunicateClient
	receiver   CentralReceiver
	finished   *sync.WaitGroup
}

var _ suite.SetupTestSuite = (*centralReceiverSuite)(nil)

var (
	testMsg = &central.MsgToSensor{
		Msg: &central.MsgToSensor_PolicySync{
			PolicySync: &central.PolicySync{},
		},
	}
	ignoredMsg = &central.MsgToSensor{
		Msg: &central.MsgToSensor_Hello{},
	}
)

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
	fastTick := make(chan struct{})
	close(fastTick)
	fastComponent := &testSensorComponent{
		t:          s.T(),
		name:       "fast",
		tick:       fastTick,
		responsesC: make(chan *message.ExpiringMessage),
	}
	blockedChan := make(chan struct{})
	blockingComponent := &testSensorComponent{
		t:          s.T(),
		name:       "blocked",
		tick:       blockedChan,
		responsesC: make(chan *message.ExpiringMessage),
	}

	components := []common.SensorComponent{fastComponent, blockingComponent}
	s.receiver = NewCentralReceiver(s.finished, components...)

	s.mockClient.EXPECT().Context().AnyTimes().Return(context.Background()).AnyTimes()

	msgCount := 3
	s.mockClient.EXPECT().Recv().MinTimes(msgCount).DoAndReturn(func() (*central.MsgToSensor, error) {
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

func (s *centralReceiverSuite) Test_FilterIgnoresMessages() {
	const (
		numberOfCentralMessages = 5
		queueSize               = 3
	)
	s.Require().NoError(os.Setenv(env.RequestsChannelBufferSize.EnvVar(), fmt.Sprintf("%d", queueSize)))
	s.T().Cleanup(func() {
		s.NoError(os.Unsetenv(env.RequestsChannelBufferSize.EnvVar()))
	})
	tick := make(chan struct{})
	fastComponent := &testSensorComponent{
		t:          s.T(),
		name:       "filtered",
		tick:       tick,
		responsesC: make(chan *message.ExpiringMessage, numberOfCentralMessages),
	}
	close(tick)

	components := []common.SensorComponent{fastComponent}
	s.receiver = NewCentralReceiver(s.finished, components...)

	// Get initial metric values before test
	initialDropped := getOpMetricValue(s.T(), fastComponent.Name(), "Drop")
	initialAddOperations := getOpMetricValue(s.T(), fastComponent.Name(), "Add")
	initialRemoveOperations := getOpMetricValue(s.T(), fastComponent.Name(), "Remove")

	s.mockClient.EXPECT().Context().AnyTimes().Return(context.Background()).AnyTimes()

	messagesFromCentral := make(chan *central.MsgToSensor, numberOfCentralMessages)
	s.mockClient.EXPECT().Recv().MinTimes(5).DoAndReturn(func() (*central.MsgToSensor, error) {
		msg, ok := <-messagesFromCentral
		if !ok {
			s.T().Logf("received EOF from central")
			return nil, io.EOF
		}
		s.T().Logf("received the message from central")
		return msg, nil
	})

	s.receiver.Start(s.mockClient)

	s.T().Logf("Sending %d messages from central", numberOfCentralMessages)
	for i := 0; i < numberOfCentralMessages; i++ {
		messagesFromCentral <- ignoredMsg
		time.Sleep(time.Millisecond)
	}
	s.T().Log("Everything should be dropped")
	s.T().Logf("Sending EOF from central")
	close(messagesFromCentral)

	s.T().Logf("Waiting for receiever to stop")
	s.finished.Wait()
	fastComponent.Stop()
	_, ok := <-fastComponent.ResponsesC()
	s.False(ok, "all message should be filtered")

	// Verify queue metrics - calculate deltas
	finalDropped := getOpMetricValue(s.T(), fastComponent.Name(), "Drop")
	finalAddOperations := getOpMetricValue(s.T(), fastComponent.Name(), "Add")
	finalRemoveOperations := getOpMetricValue(s.T(), fastComponent.Name(), "Remove")

	droppedDelta := finalDropped - initialDropped
	addDelta := finalAddOperations - initialAddOperations
	removeDelta := finalRemoveOperations - initialRemoveOperations

	s.Equal(float64(0), droppedDelta)
	s.Equal(float64(0), addDelta)
	s.Equal(float64(0), removeDelta)

	s.NoError(s.receiver.Stopped().Err())
}

func (s *centralReceiverSuite) Test_SlowComponentDropMessages() {
	const (
		numberOfCentralMessages = 5
		queueSize               = 3
	)
	s.Require().NoError(os.Setenv(env.RequestsChannelBufferSize.EnvVar(), fmt.Sprintf("%d", queueSize)))
	s.T().Cleanup(func() {
		s.NoError(os.Unsetenv(env.RequestsChannelBufferSize.EnvVar()))
	})
	tick := make(chan struct{})
	slowComponent := &testSensorComponent{
		t:          s.T(),
		name:       "slow",
		tick:       tick,
		responsesC: make(chan *message.ExpiringMessage, numberOfCentralMessages),
	}

	components := []common.SensorComponent{slowComponent}
	s.receiver = NewCentralReceiver(s.finished, components...)

	// Get initial metric values before test
	initialDropped := getOpMetricValue(s.T(), slowComponent.Name(), "Drop")
	initialAddOperations := getOpMetricValue(s.T(), slowComponent.Name(), "Add")
	initialRemoveOperations := getOpMetricValue(s.T(), slowComponent.Name(), "Remove")

	s.mockClient.EXPECT().Context().AnyTimes().Return(context.Background()).AnyTimes()

	messagesFromCentral := make(chan *central.MsgToSensor, numberOfCentralMessages)
	s.mockClient.EXPECT().Recv().MinTimes(5).DoAndReturn(func() (*central.MsgToSensor, error) {
		msg, ok := <-messagesFromCentral
		if !ok {
			s.T().Logf("received EOF from central")
			return nil, io.EOF
		}
		s.T().Logf("received the message from central")
		return msg, nil
	})

	s.receiver.Start(s.mockClient)

	s.T().Logf("Sending %d messages from central", numberOfCentralMessages)
	for i := 0; i < numberOfCentralMessages; i++ {
		messagesFromCentral <- testMsg
		time.Sleep(time.Millisecond)
	}
	s.T().Logf("Only %d messages should be processed and rest should be dropped", queueSize)
	s.T().Logf("Unblocking component")
	close(tick)

	s.T().Logf("Reading responses from component.")
	for i := 0; i <= queueSize; i++ {
		<-slowComponent.ResponsesC()
	}
	s.T().Logf("Sending EOF from central")
	close(messagesFromCentral)

	s.T().Logf("Waiting for receiever to stop")
	s.finished.Wait()
	slowComponent.Stop()
	_, ok := <-slowComponent.ResponsesC()
	s.False(ok, "no more message should be processed than what was already read")

	// Verify queue metrics - calculate deltas
	finalDropped := getOpMetricValue(s.T(), slowComponent.Name(), "Drop")
	finalAddOperations := getOpMetricValue(s.T(), slowComponent.Name(), "Add")
	finalRemoveOperations := getOpMetricValue(s.T(), slowComponent.Name(), "Remove")

	droppedDelta := finalDropped - initialDropped
	addDelta := finalAddOperations - initialAddOperations
	removeDelta := finalRemoveOperations - initialRemoveOperations

	s.Equal(float64(numberOfCentralMessages-queueSize-1), droppedDelta, "should drop excess messages beyond queue capacity")
	s.Equal(float64(numberOfCentralMessages-1), addDelta, "should track all add operations")
	s.Equal(float64(queueSize+1), removeDelta, "should track all remove operations")

	s.NoError(s.receiver.Stopped().Err())
}

func (s *centralReceiverSuite) Test_ComponentProcessMessageErrorsMetric() {
	const numberOfCentralMessages = 5
	errorComponent := &testErrorSensorComponent{testSensorComponent{
		t:    s.T(),
		name: "error-component",
	}}

	components := []common.SensorComponent{errorComponent}
	s.receiver = NewCentralReceiver(s.finished, components...)

	// Get initial error count before test
	initialErrors := metrics.GetMetricValue(s.T(), errorsMetric, map[string]string{metrics.ComponentName: "error-component"})

	s.mockClient.EXPECT().Context().AnyTimes().Return(context.Background()).AnyTimes()

	messagesFromCentral := make(chan *central.MsgToSensor)
	s.mockClient.EXPECT().Recv().MinTimes(numberOfCentralMessages).DoAndReturn(func() (*central.MsgToSensor, error) {
		msg, ok := <-messagesFromCentral
		if !ok {
			s.T().Logf("received EOF from central")
			return nil, io.EOF
		}
		s.T().Logf("received the message from central")
		return msg, nil
	})

	s.receiver.Start(s.mockClient)
	s.T().Logf("Sending %d messages from central", numberOfCentralMessages)
	for i := 0; i < numberOfCentralMessages; i++ {
		messagesFromCentral <- testMsg
	}

	s.EventuallyWithT(func(c *assert.CollectT) {
		finalErrors := metrics.GetMetricValue(s.T(), errorsMetric, map[string]string{metrics.ComponentName: "error-component"})
		assert.Equal(c, numberOfCentralMessages, int(finalErrors-initialErrors))
	}, 5*time.Second, 10*time.Millisecond, "error metric should be incremented when ProcessMessage returns an error")

	s.T().Logf("Sending EOF from central")
	close(messagesFromCentral)

	s.T().Logf("Waiting for receiever to stop")
	s.finished.Wait()
	s.NoError(s.receiver.Stopped().Err())
}

// testSensorComponent process messages with every tick
type testSensorComponent struct {
	t          *testing.T
	name       string
	tick       <-chan struct{}
	responsesC chan *message.ExpiringMessage
}

func (t *testSensorComponent) Filter(msg *central.MsgToSensor) bool {
	return msg.GetPolicySync() != nil
}

func (t *testSensorComponent) ProcessMessage(ctx context.Context, msg *central.MsgToSensor) error {
	require.NotNil(t.t, msg.GetPolicySync(), "should have received policy sync")
	select {
	case <-t.tick:
		select {
		case t.responsesC <- &message.ExpiringMessage{}:
			t.t.Logf("%s: message processed", t.Name())
			return nil
		case <-ctx.Done():
			t.t.Logf("%s: %s", t.Name(), ctx.Err())
			return ctx.Err()
		}
	case <-ctx.Done():
		t.t.Logf("%s: %s", t.Name(), ctx.Err())
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

// testErrorSensorComponent always returns an error from ProcessMessage
type testErrorSensorComponent struct {
	testSensorComponent
}

func (t *testErrorSensorComponent) ProcessMessage(_ context.Context, _ *central.MsgToSensor) error {
	t.t.Logf("%s: returning error", t.Name())
	return errors.New("test error")
}

func getOpMetricValue(t *testing.T, componet, op string) float64 {
	t.Helper()
	return metrics.GetMetricValue(t, totalMetric, map[string]string{metrics.ComponentName: componet, metrics.Operation: op})
}
