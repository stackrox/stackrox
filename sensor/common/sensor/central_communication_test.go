package sensor

import (
	"context"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	configMocks "github.com/stackrox/rox/sensor/common/config/mocks"
	mocksDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/message"
	mocksClient "github.com/stackrox/rox/sensor/common/sensor/mocks"
	debuggerMessage "github.com/stackrox/rox/sensor/debugger/message"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type centralCommunicationSuite struct {
	suite.Suite

	controller   *gomock.Controller
	mockHandler  *configMocks.MockHandler
	mockDetector *mocksDetector.MockDetector
	mockService  *MockSensorServiceClient
	comm         CentralCommunication
	responsesC   chan *message.ExpiringMessage
}

var _ suite.SetupTestSuite = (*centralCommunicationSuite)(nil)
var _ suite.TearDownTestSuite = (*centralCommunicationSuite)(nil)

func (c *centralCommunicationSuite) SetupTest() {
	mockCtrl := gomock.NewController(c.T())

	c.controller = mockCtrl
	c.mockHandler = configMocks.NewMockHandler(mockCtrl)
	c.mockDetector = mocksDetector.NewMockDetector(mockCtrl)

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

	// Create a fake SensorComponent
	c.responsesC = make(chan *message.ExpiringMessage)
	c.comm = NewCentralCommunication(false, NewFakeSensorComponent(c.responsesC))

	c.mockService = &MockSensorServiceClient{
		connected: concurrency.NewSignal(),
		client:    mocksClient.NewMockServiceCommunicateClient(c.controller),
	}
}

func (c *centralCommunicationSuite) TearDownTest() {
	defer close(c.responsesC)
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
}

func (c *centralCommunicationSuite) Test_StartCentralCommunication() {
	expectSyncMessages(centralSyncMessages, c.mockService)
	ch := make(chan struct{})
	c.mockService.client.EXPECT().Send(gomock.Any()).Times(1).DoAndReturn(func(msg *central.MsgFromSensor) error {
		defer close(ch)
		c.Assert().NotNil(msg.GetEvent().GetSynced())
		return nil
	})

	reachable := concurrency.Flag{}
	// Start the go routine with the mocked client
	go c.comm.(*centralCommunicationImpl).sendEvents(c.mockService, &reachable, c.mockHandler, c.mockDetector)
	c.mockService.connected.Wait()

	// Pretend that a component (listener) is sending the sync event
	c.responsesC <- message.New(syncMessage())
	select {
	case <-ch:
		break
	case <-time.After(5 * time.Second):
		c.Fail("timeout reached waiting for the sync event")
	}
}

func (c *centralCommunicationSuite) Test_StopCentralCommunication() {
	expectSyncMessages(centralSyncMessages, c.mockService)
	ch := make(chan struct{})
	c.mockService.client.EXPECT().CloseSend().Times(1).DoAndReturn(func() error {
		defer close(ch)
		return nil
	})

	reachable := concurrency.Flag{}
	// Start the go routine with the mocked client
	go c.comm.(*centralCommunicationImpl).sendEvents(c.mockService, &reachable, c.mockHandler, c.mockDetector, c.comm.(*centralCommunicationImpl).receiver.Stop, c.comm.(*centralCommunicationImpl).sender.Stop)
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

func expectSyncMessages(messages []*central.MsgToSensor, service *MockSensorServiceClient) {
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
	orderedCalls = append(orderedCalls, service.client.EXPECT().Recv().AnyTimes())
	gomock.InOrder(orderedCalls...)
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
