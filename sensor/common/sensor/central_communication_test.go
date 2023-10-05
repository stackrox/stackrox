package sensor

import (
	"context"
	"fmt"
	"net"
	"path"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/sensor/hash"
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

	receivedMessages     chan *central.MsgFromSensor
	conn                 *grpc.ClientConn
	centralCommunication CentralCommunication
	closeF               func()
	mockHandler          *configMocks.MockHandler
	mockDetector         *mocksDetector.MockDetector
	fakeCentral          *centralDebug.FakeService
}

var _ suite.SetupTestSuite = (*centralCommunicationSuite)(nil)
var _ suite.TearDownTestSuite = (*centralCommunicationSuite)(nil)

func (c *centralCommunicationSuite) TearDownTest() {
	c.fakeCentral.Stop()
	c.closeF()
}

func (c *centralCommunicationSuite) SetupTest() {
	mockCtrl := gomock.NewController(c.T())

	c.mockHandler = configMocks.NewMockHandler(mockCtrl)
	c.mockDetector = mocksDetector.NewMockDetector(mockCtrl)

	c.receivedMessages = make(chan *central.MsgFromSensor, 10)

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
}

func Test_CentralCommunicationSuite(t *testing.T) {
	suite.Run(t, new(centralCommunicationSuite))
}

func (c *centralCommunicationSuite) Test_CentralCommunication_Start() {
	c.setupFakeCentral(c.serverReconciliationFakeCentral())
	componentMessages := c.givenCentralCommunication()

	// Pretend that a component (listener) is sending the sync event
	componentMessages <- message.New(syncMessage())

	// Wait for sync message or timeout after 5s
	timeout := time.After(5 * time.Second)
	var syncReceived bool
	for syncReceived {
		select {
		case <-timeout:
			c.Fail("Didn't receive sync message after 5 seconds")
		case msg := <-c.receivedMessages:
			syncReceived = msg.GetEvent().GetSynced() != nil
		}
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
		deduperState           map[string]uint64
		componentMessages      []*central.MsgFromSensor
		neverMatchesState      func(messages []*central.MsgFromSensor) bool
		eventuallyMatchesState func(messages []*central.MsgFromSensor) bool
	}{
		"Deduper hash hit": {
			deduperState: map[string]uint64{
				deploymentKey(fixtureconsts.Deployment1): hash1,
			},
			componentMessages: []*central.MsgFromSensor{dep1},
			neverMatchesState: anyDeploymentSent,
		},
		"All deployments sent": {
			deduperState:           map[string]uint64{},
			componentMessages:      []*central.MsgFromSensor{dep1, dep2},
			eventuallyMatchesState: deploymentsSentCount(2),
		},
		"Updated deployment": {
			deduperState: map[string]uint64{
				deploymentKey(fixtureconsts.Deployment1): hash1,
			},
			componentMessages:      []*central.MsgFromSensor{updatedDep1},
			eventuallyMatchesState: deploymentIDSent(fixtureconsts.Deployment1, updatedHash1),
		},
	}

	for name, tc := range testCases {
		c.Run(name, func() {
			c.setupFakeCentral(c.clientReconciliationFakeCentral(tc.deduperState))
			componentMessages := c.givenCentralCommunication()

			for _, msg := range tc.componentMessages {
				componentMessages <- message.New(msg)
			}

			if tc.eventuallyMatchesState != nil {
				c.Assert().Eventually(func() bool {
					return tc.eventuallyMatchesState(c.fakeCentral.GetAllMessages())
				}, 3*time.Second, 100*time.Millisecond)
			}

			if tc.neverMatchesState != nil {
				c.Assert().Never(func() bool {
					return tc.neverMatchesState(c.fakeCentral.GetAllMessages())
				}, 3*time.Second, 100*time.Millisecond)
			}
		})
	}
}

func deploymentIDSent(id string, hash uint64) func([]*central.MsgFromSensor) bool {
	return func(messages []*central.MsgFromSensor) bool {
		for _, m := range messages {
			if dep := m.GetEvent().GetDeployment(); dep != nil {
				if dep.GetId() == id && m.GetEvent().GetSensorHash() == hash {
					return true
				}
			}
		}
		return false
	}
}

func deploymentsSentCount(n int) func([]*central.MsgFromSensor) bool {
	return func(messages []*central.MsgFromSensor) bool {
		var count int
		for _, m := range messages {
			if m.GetEvent().GetDeployment() != nil {
				count += 1
			}
		}
		return count == n
	}
}

func anyDeploymentSent(messages []*central.MsgFromSensor) bool {
	for _, m := range messages {
		if m.GetEvent().GetDeployment() != nil {
			return true
		}
	}
	return false
}
func deploymentKey(id string) string {
	return fmt.Sprintf("Deployment:%s", id)
}

func NewFakeSensorComponent(responsesC chan *message.ExpiringMessage) common.SensorComponent {
	return &fakeSensorComponent{
		responsesC: responsesC,
	}
}

func (c *centralCommunicationSuite) setupFakeCentral(fc *centralDebug.FakeService) {
	c.fakeCentral = fc

	c.fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
		c.receivedMessages <- msg
	})

	c.conn, c.closeF = createConnectionAndStartServer(c.fakeCentral)
}

func (c *centralCommunicationSuite) clientReconciliationFakeCentral(deduperState map[string]uint64) *centralDebug.FakeService {
	return centralDebug.MakeFakeCentralWithInitialMessages(
		debuggerMessage.SensorHello("00000000-0000-4000-A000-000000000000"),
		debuggerMessage.ClusterConfig(),
		debuggerMessage.PolicySync([]*storage.Policy{}),
		debuggerMessage.BaselineSync([]*storage.ProcessBaseline{}),
		debuggerMessage.DeduperState(deduperState))
}

func (c *centralCommunicationSuite) serverReconciliationFakeCentral() *centralDebug.FakeService {
	return centralDebug.MakeFakeCentralWithInitialMessages(
		debuggerMessage.SensorHello("00000000-0000-4000-A000-000000000000"),
		debuggerMessage.ClusterConfig(),
		debuggerMessage.PolicySync([]*storage.Policy{}),
		debuggerMessage.BaselineSync([]*storage.ProcessBaseline{}))
}

func (c *centralCommunicationSuite) givenCentralCommunication() chan *message.ExpiringMessage {
	componentMessages := make(chan *message.ExpiringMessage, 10)
	c.centralCommunication = c.startCentralCommunication(componentMessages)
	return componentMessages
}

func (c *centralCommunicationSuite) startCentralCommunication(componentMessages chan *message.ExpiringMessage) CentralCommunication {
	comms := NewCentralCommunication(true, false, NewFakeSensorComponent(componentMessages))

	reachable := concurrency.Flag{}
	// This implicitly starts a goroutine
	comms.Start(c.conn, &reachable, c.mockHandler, c.mockDetector)
	return comms
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
