package sensor

import (
	"context"
	"fmt"
	"path"
	"strings"
	"testing"
	"time"

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
	c.comm = NewCentralCommunication(true, false, NewFakeSensorComponent(c.responsesC))

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
	debuggerMessage.NetworkBaselineSync([]*storage.NetworkBaseline{}),
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
	var orderedCalls []*gomock.Call
	for _, m := range messages {
		orderedCalls = append(orderedCalls, service.client.EXPECT().Recv().Times(1).Return(m, nil))
	}
	orderedCalls = append(orderedCalls, service.client.EXPECT().Recv().AnyTimes())
	gomock.InOrder(orderedCalls...)
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
			syncMessages := append(centralSyncMessages, debuggerMessage.DeduperState(tc.deduperState))
			expectSyncMessages(syncMessages, c.mockService)

			var sentMessages []*central.MsgFromSensor
			c.mockService.client.EXPECT().Send(gomock.Any()).AnyTimes().DoAndReturn(func(msg *central.MsgFromSensor) error {
				sentMessages = append(sentMessages, msg)
				return nil
			})

			reachable := concurrency.Flag{}
			// Start the go routine with the mocked client
			go c.comm.(*centralCommunicationImpl).sendEvents(c.mockService, &reachable, c.mockHandler, c.mockDetector, c.comm.(*centralCommunicationImpl).receiver.Stop, c.comm.(*centralCommunicationImpl).sender.Stop)
			c.mockService.connected.Wait()

			for _, msg := range tc.componentMessages {
				c.responsesC <- message.New(msg)
			}

			if tc.eventuallyMatchesState != nil {
				c.Assert().Eventually(func() bool {
					return tc.eventuallyMatchesState(sentMessages)
				}, 3*time.Second, 100*time.Millisecond)
			}

			if tc.neverMatchesState != nil {
				c.Assert().Never(func() bool {
					return tc.neverMatchesState(sentMessages)
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
		if count == n {
			return true
		}
		return false
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
