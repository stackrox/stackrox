package sensor

import (
	"context"
	"fmt"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/sensor/hash"
	"github.com/stackrox/rox/sensor/common"
	configMocks "github.com/stackrox/rox/sensor/common/config/mocks"
	mocksDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/message"
	mocksClient "github.com/stackrox/rox/sensor/common/sensor/mocks"
	storeMock "github.com/stackrox/rox/sensor/common/store/mocks"
	debuggerMessage "github.com/stackrox/rox/sensor/debugger/message"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type centralCommunicationSuite struct {
	suite.Suite

	controller     *gomock.Controller
	mockHandler    *configMocks.MockHandler
	mockDetector   *mocksDetector.MockDetector
	mockService    *MockSensorServiceClient
	mockReconciler *storeMock.MockHashReconciler
	comm           CentralCommunication
}

var _ suite.SetupTestSuite = (*centralCommunicationSuite)(nil)

func (c *centralCommunicationSuite) SetupTest() {
	mockCtrl := gomock.NewController(c.T())

	c.controller = mockCtrl
	c.mockHandler = configMocks.NewMockHandler(mockCtrl)
	c.mockDetector = mocksDetector.NewMockDetector(mockCtrl)
	c.mockReconciler = storeMock.NewMockHashReconciler(mockCtrl)

	certPath := "../../../tools/local-sensor/certs/"

	c.T().Setenv("ROX_MTLS_CERT_FILE", path.Join(certPath, "/cert.pem"))
	c.T().Setenv("ROX_MTLS_KEY_FILE", path.Join(certPath, "/key.pem"))
	c.T().Setenv("ROX_MTLS_CA_FILE", path.Join(certPath, "/caCert.pem"))
	c.T().Setenv("ROX_MTLS_CA_KEY_FILE", path.Join(certPath, "/caKey.pem"))

	// Setup Mocks:
	c.mockHandler.EXPECT().GetDeploymentIdentification().AnyTimes().Return(nil)
	c.mockHandler.EXPECT().GetHelmManagedConfig().AnyTimes().Return(nil)
	c.mockHandler.EXPECT().ProcessMessage(gomock.Any()).AnyTimes().Return(nil)
	c.mockDetector.EXPECT().ProcessMessage(gomock.Any()).AnyTimes().Return(nil)
	c.mockDetector.EXPECT().ProcessPolicySync(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	c.mockReconciler.EXPECT().ProcessHashes(gomock.Any()).AnyTimes().Return(nil)
}

func Test_CentralCommunicationSuite(t *testing.T) {
	suite.Run(t, new(centralCommunicationSuite))
}

type MockSensorServiceClient struct {
	client    *mocksClient.MockServiceCommunicateClient
	connected concurrency.Signal
}

func (s *MockSensorServiceClient) Communicate(_ context.Context, _ ...grpc.CallOption) (central.SensorService_CommunicateClient, error) {
	defer s.connected.Signal()
	return s.client, nil
}

var _ central.SensorServiceClient = (*MockSensorServiceClient)(nil)

var centralSyncMessages = []*central.MsgToSensor{
	debuggerMessage.SensorHello("00000000-0000-4000-A000-000000000000"),
	debuggerMessage.ClusterConfig(),
	debuggerMessage.PolicySync([]*storage.Policy{}),
	debuggerMessage.BaselineSync([]*storage.ProcessBaseline{}),
	debuggerMessage.NetworkBaselineSync([]*storage.NetworkBaseline{}),
}

func (c *centralCommunicationSuite) Test_StartCentralCommunication() {
	responsesC, closeFn := c.createCentralCommunication(false)
	defer closeFn()
	expectSyncMessagesNoBlockRecv(centralSyncMessages, c.mockService)
	ch := make(chan struct{})
	c.mockService.client.EXPECT().Send(gomock.Any()).Times(1).DoAndReturn(func(msg *central.MsgFromSensor) error {
		defer close(ch)
		c.Assert().NotNil(msg.GetEvent().GetSynced())
		return nil
	})

	reachable := concurrency.Flag{}
	// Start the go routine with the mocked client
	c.comm.Start(c.mockService, &reachable, c.mockHandler, c.mockDetector)
	c.mockService.connected.Wait()

	// Pretend that a component (listener) is sending the sync event
	responsesC <- message.New(syncMessage())
	select {
	case <-ch:
		break
	case <-time.After(5 * time.Second):
		c.Fail("timeout reached waiting for the sync event")
	}
}

func (c *centralCommunicationSuite) Test_StopCentralCommunication() {
	_, closeFn := c.createCentralCommunication(false)
	defer closeFn()
	expectSyncMessagesNoBlockRecv(centralSyncMessages, c.mockService)
	ch := make(chan struct{})
	c.mockService.client.EXPECT().CloseSend().Times(1).DoAndReturn(func() error {
		defer close(ch)
		return nil
	})

	reachable := concurrency.Flag{}
	// Start the go routine with the mocked client
	c.comm.Start(c.mockService, &reachable, c.mockHandler, c.mockDetector)
	c.mockService.connected.Wait()

	// Stop CentralCommunication
	c.comm.Stop(nil)
	select {
	case <-ch:
		break
	case <-time.After(5 * time.Second):
		c.Fail("timeout reached waiting for the communication to stop")
	}
}

func (c *centralCommunicationSuite) Test_ClientReconciliation() {
	dep1 := givenDeployment(fixtureconsts.Deployment1, "dep1", map[string]string{"app": "central"})
	dep2 := givenDeployment(fixtureconsts.Deployment2, "dep2", map[string]string{"app": "sensor"})
	updatedDep1 := givenDeployment(fixtureconsts.Deployment1, "dep1", map[string]string{"app": "central_updated"})

	hasher := hash.NewHasher()
	hash1, _ := hasher.HashEvent(dep1.GetEvent())
	updatedHash1, _ := hasher.HashEvent(updatedDep1.GetEvent())
	setSensorHash(dep1, hash1)
	setSensorHash(updatedDep1, updatedHash1)

	testCases := map[string]struct {
		deduperState      map[string]uint64
		componentMessages []*central.MsgFromSensor
		expectedMessages  *messagesMatcher
	}{
		"Deduper hash hit": {
			deduperState: map[string]uint64{
				deploymentKey(fixtureconsts.Deployment1): hash1,
			},
			componentMessages: []*central.MsgFromSensor{dep1},
			expectedMessages:  newMessagesMatcher("no deployment should be sent"),
		},
		"Deduper hash hit in second event": {
			deduperState:      map[string]uint64{},
			componentMessages: []*central.MsgFromSensor{dep1, dep1},
			expectedMessages:  newMessagesMatcher("first deployment should be sent", dep1),
		},
		"All deployments sent": {
			deduperState:      map[string]uint64{},
			componentMessages: []*central.MsgFromSensor{dep1, dep2},
			expectedMessages:  newMessagesMatcher("all deployments should be sent", dep1, dep2),
		},
		"Updated deployment": {
			deduperState: map[string]uint64{
				deploymentKey(fixtureconsts.Deployment1): hash1,
			},
			componentMessages: []*central.MsgFromSensor{updatedDep1},
			expectedMessages:  newMessagesMatcher("updated deployment should be sent", updatedDep1),
		},
	}

	for name, tc := range testCases {
		c.Run(name, func() {
			responsesC, closeFn := c.createCentralCommunication(true)
			defer closeFn()
			syncMessages := append(centralSyncMessages, debuggerMessage.DeduperState(tc.deduperState))
			expectSyncMessagesNoBlockRecv(syncMessages, c.mockService)

			c.mockService.client.EXPECT().Send(tc.expectedMessages).Times(len(tc.expectedMessages.messagesToMatch))
			c.mockService.client.EXPECT().CloseSend().AnyTimes()

			reachable := concurrency.Flag{}
			// Start the go routine with the mocked client
			c.comm.Start(c.mockService, &reachable, c.mockHandler, c.mockDetector)
			c.mockService.connected.Wait()

			for _, msg := range tc.componentMessages {
				responsesC <- message.New(msg)
			}

			select {
			case <-time.After(5 * time.Second):
				c.Failf("timeout waiting for test state", tc.expectedMessages.error)
			case <-tc.expectedMessages.matcherIsDone.Done():
				break
			}
		})
	}
}

func (c *centralCommunicationSuite) Test_TimeoutWaitingForDeduperState() {
	_, closeFn := c.createCentralCommunication(true)
	defer closeFn()
	recvSignal := expectSyncMessages(centralSyncMessages, true, c.mockService)
	ch := make(chan struct{})
	c.mockService.client.EXPECT().CloseSend().Times(1).DoAndReturn(func() error {
		defer close(ch)
		return nil
	})

	reachable := concurrency.Flag{}
	c.comm.(*centralCommunicationImpl).syncTimeout = 10 * time.Millisecond
	// Start the go routine with the mocked client
	c.comm.Start(c.mockService, &reachable, c.mockHandler, c.mockDetector)
	c.mockService.connected.Wait()

	select {
	case <-ch:
		c.Assert().ErrorIs(c.comm.Stopped().Err(), errTimeoutWaitingForDeduperState)
		break
	case <-time.After(5 * time.Second):
		c.Fail("timeout reached waiting for the connection to timeout if the deduper state is not received")
	}
	recvSignal.Signal()
}

type messagesMatcher struct {
	messagesToMatch map[string]*central.MsgFromSensor
	cmpFn           func(x, y *central.MsgFromSensor) bool
	matcherIsDone   concurrency.Signal
	error           string
}

func (m *messagesMatcher) Matches(target interface{}) bool {
	msg, ok := target.(*central.MsgFromSensor)
	if !ok {
		m.error += " received message that isn't a MsgFromSensor"
		return false
	}
	if expectedMsg, found := m.messagesToMatch[msg.GetEvent().GetId()]; found && m.cmpFn(expectedMsg, msg) {
		delete(m.messagesToMatch, msg.GetEvent().GetId())
		if len(m.messagesToMatch) == 0 {
			// We are done processing the expected messages
			m.matcherIsDone.Signal()
		}
		return true
	}
	m.error += fmt.Sprintf(" unexpected event: %+v", msg.GetEvent())
	return false
}

func (m *messagesMatcher) String() string {
	return fmt.Sprintf("expected %v: error: %s", m.messagesToMatch, m.error)
}

func newMessagesMatcher(errorMsg string, msgs ...*central.MsgFromSensor) *messagesMatcher {
	ret := &messagesMatcher{
		messagesToMatch: make(map[string]*central.MsgFromSensor),
		matcherIsDone:   concurrency.NewSignal(),
		error:           errorMsg,
		cmpFn: func(x, y *central.MsgFromSensor) bool {
			if x.GetEvent() == nil || y.GetEvent() == nil {
				return false
			}
			return x.GetEvent().GetId() == y.GetEvent().GetId() && cmp.Equal(x.GetEvent().GetDeployment(), y.GetEvent().GetDeployment())
		},
	}
	for _, m := range msgs {
		ret.messagesToMatch[m.GetEvent().GetId()] = m
	}
	if len(msgs) == 0 {
		// If we are not expecting any messages we can go ahead and trigger the signal.
		// The test will fail if any messages are sent since the mock expects Send to be called 0 times.
		ret.matcherIsDone.Signal()
	}
	return ret
}

func (c *centralCommunicationSuite) createCentralCommunication(clientReconcile bool) (chan *message.ExpiringMessage, func()) {
	// Create a CentralCommunication with a fake SensorComponent
	ret := make(chan *message.ExpiringMessage)
	c.comm = NewCentralCommunication(nil, false, clientReconcile, NewFakeSensorComponent(ret))
	// Initialize the gRPC mocked service
	c.mockService = &MockSensorServiceClient{
		connected: concurrency.NewSignal(),
		client:    mocksClient.NewMockServiceCommunicateClient(c.controller),
	}
	return ret, func() { close(ret) }
}

func expectSyncMessagesNoBlockRecv(messages []*central.MsgToSensor, service *MockSensorServiceClient) {
	_ = expectSyncMessages(messages, false, service)
}

func expectSyncMessages(messages []*central.MsgToSensor, blockRecv bool, service *MockSensorServiceClient) concurrency.Signal {
	signal := concurrency.NewSignal()
	md := metadata.MD{
		strings.ToLower(centralsensor.SensorHelloMetadataKey): []string{"true"},
	}
	service.client.EXPECT().Header().AnyTimes().Return(md, nil)
	service.client.EXPECT().Send(gomock.Any()).Return(nil)
	service.client.EXPECT().Context().AnyTimes().Return(context.Background())
	var orderedCalls []any
	for _, m := range messages {
		orderedCalls = append(orderedCalls, service.client.EXPECT().Recv().Times(1).Return(m, nil))
	}
	if !blockRecv {
		orderedCalls = append(orderedCalls, service.client.EXPECT().Recv().AnyTimes())
	} else {
		orderedCalls = append(orderedCalls, service.client.EXPECT().Recv().AnyTimes().DoAndReturn(func() (*central.MsgToSensor, error) {
			// This will block the Recv() calls until the signal is triggered. Otherwise, we process constantly Recv()
			signal.Wait()
			return nil, nil
		}))
	}
	gomock.InOrder(orderedCalls...)
	return signal
}

func deploymentKey(id string) string {
	return fmt.Sprintf("Deployment:%s", id)
}

func givenDeployment(uuid, name string, labels map[string]string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		HashKey:           "",
		DedupeKey:         "",
		ProcessingAttempt: 0,
		Msg: &central.MsgFromSensor_Event{
			// The hash in the gRPC deduper is constructed by the central.SensorEvent struct
			// Any changes in this struct will prevent the deduper from filtering the message
			Event: &central.SensorEvent{
				Id:     uuid,
				Action: 0,
				Timing: nil,
				// SensorHash stores the SensorEvent hash. It is set later
				SensorHashOneof: nil,
				Resource: &central.SensorEvent_Deployment{
					Deployment: &storage.Deployment{
						Id:     uuid,
						Name:   name,
						Labels: labels,
					},
				},
			},
		},
	}
}

func setSensorHash(sensorMsg *central.MsgFromSensor, sensorHash uint64) {
	if event := sensorMsg.GetEvent(); event != nil {
		event.SensorHashOneof = &central.SensorEvent_SensorHash{
			SensorHash: sensorHash,
		}
	}
}

func NewFakeSensorComponent(responsesC chan *message.ExpiringMessage) common.SensorComponent {
	return &fakeSensorComponent{
		responsesC: responsesC,
	}
}

func syncMessage() *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_Synced{
					Synced: &central.SensorEvent_ResourcesSynced{},
				},
			},
		},
	}
}

type fakeSensorComponent struct {
	responsesC chan *message.ExpiringMessage
}

func (f fakeSensorComponent) Notify(common.SensorComponentEvent) {
	panic("implement me")
}

func (f fakeSensorComponent) Start() error {
	panic("implement me")
}

func (f fakeSensorComponent) Stop(error) {
	panic("implement me")
}

func (f fakeSensorComponent) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{}
}

func (f fakeSensorComponent) ProcessMessage(*central.MsgToSensor) error {
	return nil
}

func (f fakeSensorComponent) ResponsesC() <-chan *message.ExpiringMessage {
	return f.responsesC
}

var _ common.SensorComponent = (*fakeSensorComponent)(nil)
