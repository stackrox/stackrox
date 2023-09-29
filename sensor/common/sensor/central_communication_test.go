package sensor

import (
	"context"
	"net"
	"path"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	configMocks "github.com/stackrox/rox/sensor/common/config/mocks"
	mocksDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/message"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	debuggerMessage "github.com/stackrox/rox/sensor/debugger/message"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type centralCommunicationSuite struct {
	suite.Suite

	receivedMessages chan *central.MsgFromSensor
	conn             *grpc.ClientConn
	closeF           func()
	mockHandler      *configMocks.MockHandler
	mockDetector     *mocksDetector.MockDetector
	fakeCentral      *centralDebug.FakeService
}

var _ suite.SetupTestSuite = (*centralCommunicationSuite)(nil)
var _ suite.TearDownTestSuite = (*centralCommunicationSuite)(nil)

func (c *centralCommunicationSuite) TearDownTest() {
	c.fakeCentral.Stop()
	c.fakeCentral.ServerPointer.GracefulStop()
	c.closeF()
}

func (c *centralCommunicationSuite) SetupTest() {
	mockCtrl := gomock.NewController(c.T())

	c.mockHandler = configMocks.NewMockHandler(mockCtrl)
	c.mockDetector = mocksDetector.NewMockDetector(mockCtrl)

	c.receivedMessages = make(chan *central.MsgFromSensor, 10)

	c.fakeCentral = centralDebug.MakeFakeCentralWithInitialMessages(
		debuggerMessage.SensorHello("00000000-0000-4000-A000-000000000000"),
		debuggerMessage.ClusterConfig(),
		debuggerMessage.PolicySync([]*storage.Policy{}),
		debuggerMessage.BaselineSync([]*storage.ProcessBaseline{}))

	c.fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
		c.receivedMessages <- msg
	})

	certPath := "../../../tools/local-sensor/certs/"

	c.T().Setenv("ROX_MTLS_CERT_FILE", path.Join(certPath, "/cert.pem"))
	c.T().Setenv("ROX_MTLS_KEY_FILE", path.Join(certPath, "/key.pem"))
	c.T().Setenv("ROX_MTLS_CA_FILE", path.Join(certPath, "/caCert.pem"))
	c.T().Setenv("ROX_MTLS_CA_KEY_FILE", path.Join(certPath, "/caKey.pem"))

	c.conn, c.closeF = createConnectionAndStartServer(c.fakeCentral)

	// Setup Mocks:
	c.mockHandler.EXPECT().GetDeploymentIdentification().AnyTimes().Return(nil)
	c.mockHandler.EXPECT().GetHelmManagedConfig().AnyTimes().Return(nil)
	c.mockHandler.EXPECT().ProcessMessage(gomock.Any()).AnyTimes().Return(nil)
	c.mockDetector.EXPECT().ProcessMessage(gomock.Any()).AnyTimes().Return(nil)
	c.mockDetector.EXPECT().ProcessPolicySync(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
}

func Test_CentralCommunicationSuite(t *testing.T) {
	suite.Run(t, new(centralCommunicationSuite))
}

func (c *centralCommunicationSuite) Test_StartCentralCommunication() {
	componentMessages := c.givenCentralCommunication()

	// Pretend that a component (listener) is sending the sync event
	componentMessages <- message.New(syncMessage())

	// Wait for sync message or timeout after 5s
	timeout := time.After(5 * time.Second)
	var syncReceived bool
	for syncReceived {
		select {
		case <-timeout:
			c.Fail("Didn't receive sync message after 3 seconds")
		case msg := <-c.receivedMessages:
			c.T().Logf("Received: %s", msg.String())
			syncReceived = msg.GetEvent().GetSynced() != nil
		}
	}
}

func NewFakeSensorComponent(responsesC chan *message.ExpiringMessage) common.SensorComponent {
	return &fakeSensorComponent{
		responsesC: responsesC,
	}
}

func (c *centralCommunicationSuite) givenCentralCommunication() chan *message.ExpiringMessage {
	componentMessages := make(chan *message.ExpiringMessage, 10)
	comms := NewCentralCommunication(false, NewFakeSensorComponent(componentMessages))

	reachable := concurrency.Flag{}
	// This implicitly starts a goroutine
	comms.Start(c.conn, &reachable, c.mockHandler, c.mockDetector)

	return componentMessages
}

func createConnectionAndStartServer(fakeCentral *centralDebug.FakeService) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	fakeCentral.ServerPointer = grpc.NewServer()
	central.RegisterSensorServiceServer(fakeCentral.ServerPointer, fakeCentral)

	go func() {
		utils.IgnoreError(func() error {
			return fakeCentral.ServerPointer.Serve(listener)
		})
	}()

	conn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		panic(err)
	}

	closeF := func() {
		utils.IgnoreError(listener.Close)
		fakeCentral.ServerPointer.Stop()
	}

	return conn, closeF
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
	panic("implement me")
}

func (f fakeSensorComponent) ResponsesC() <-chan *message.ExpiringMessage {
	return f.responsesC
}

var _ common.SensorComponent = (*fakeSensorComponent)(nil)
