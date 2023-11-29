package eventpipeline

import (
	"sync/atomic"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	mockDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/message"
	mockReprocessor "github.com/stackrox/rox/sensor/common/reprocessor/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	mockComponent "github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type eventPipelineSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	resolver    *mockComponent.MockResolver
	detector    *mockDetector.MockDetector
	reprocessor *mockReprocessor.MockHandler
	listener    *mockComponent.MockContextListener
	outputQueue *mockComponent.MockOutputQueue
	pipeline    *eventPipeline

	outputC chan *message.ExpiringMessage
}

var _ suite.SetupTestSuite = &eventPipelineSuite{}
var _ suite.TearDownTestSuite = &eventPipelineSuite{}

func TestEventPipelineSuite(t *testing.T) {
	suite.Run(t, new(eventPipelineSuite))
}

func (s *eventPipelineSuite) TearDownTest() {
	s.T().Cleanup(s.mockCtrl.Finish)
}

func (s *eventPipelineSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.resolver = mockComponent.NewMockResolver(s.mockCtrl)
	s.detector = mockDetector.NewMockDetector(s.mockCtrl)
	s.reprocessor = mockReprocessor.NewMockHandler(s.mockCtrl)
	s.listener = mockComponent.NewMockContextListener(s.mockCtrl)
	s.outputQueue = mockComponent.NewMockOutputQueue(s.mockCtrl)

	offlineMode := atomic.Bool{}
	offlineMode.Store(true)

	s.pipeline = &eventPipeline{
		eventsC:     make(chan *message.ExpiringMessage),
		stopSig:     concurrency.NewSignal(),
		output:      s.outputQueue,
		resolver:    s.resolver,
		detector:    s.detector,
		reprocessor: s.reprocessor,
		listener:    s.listener,
		offlineMode: &offlineMode,
	}
}

func (s *eventPipelineSuite) write() {
	s.outputC <- message.NewExpiring(s.pipeline.context, nil)
}

func (s *eventPipelineSuite) online() {
	s.pipeline.Notify(common.SensorComponentEventCentralReachable)
}

func (s *eventPipelineSuite) offline() {
	s.pipeline.Notify(common.SensorComponentEventOfflineMode)
}

func (s *eventPipelineSuite) readSuccess() {
	msg, more := <-s.pipeline.ResponsesC()
	s.Assert().True(more, "channel should be open")
	s.Assert().False(msg.IsExpired(), "message should not be expired")
}

func (s *eventPipelineSuite) readExpired() {
	msg, more := <-s.pipeline.ResponsesC()
	s.Assert().True(more, "channel should be open")
	s.Assert().True(msg.IsExpired(), "message should be expired")
}

func (s *eventPipelineSuite) Test_OfflineModeCases() {
	outputC := make(chan *message.ExpiringMessage, 10)
	s.outputQueue.EXPECT().ResponsesC().
		AnyTimes().Return(outputC)
	s.outputC = outputC

	s.outputQueue.EXPECT().Start().Times(1)
	s.resolver.EXPECT().Start().Times(1)
	s.listener.EXPECT().StartWithContext(gomock.Any()).AnyTimes()
	s.listener.EXPECT().Stop(gomock.Any()).AnyTimes()

	s.Require().NoError(s.pipeline.Start())
	s.pipeline.Notify(common.SensorComponentEventCentralReachable)

	testCases := map[string][]func(){
		"Base case: Start, WA, WB, RA, RB, Disconnect":       {s.online, s.write, s.write, s.readSuccess, s.readSuccess, s.offline},
		"Case: Start, WA, WB, Disconnect, RA, RB, Reconnect": {s.write, s.write, s.offline, s.readExpired, s.readExpired, s.online},
		"Case: Start, WA, WB, Disconnect, Reconnect, RA, RB": {s.write, s.write, s.offline, s.online, s.readExpired, s.readExpired},
		"Case: Start, WA, Disconnect, WB, Reconnect, RA, RB": {s.write, s.offline, s.write, s.online, s.readExpired, s.readExpired},
		"Case: Start, WA, Disconnect, Reconnect, WB, RA, RB": {s.write, s.offline, s.online, s.write, s.readExpired, s.readSuccess},
		"Case: Start, Disconnect, WA, Reconnect, WB, RA, RB": {s.offline, s.write, s.online, s.write, s.readExpired, s.readSuccess},
		"Case: Start, Disconnect, Reconnect, WA, WB, RA, RB": {s.offline, s.online, s.write, s.write, s.readSuccess, s.readSuccess},
	}

	for caseName, orderedFunctions := range testCases {
		s.Run(caseName, func() {
			for _, fn := range orderedFunctions {
				fn()
			}
		})
	}
}

func (s *eventPipelineSuite) Test_OfflineMode() {
	outputC := make(chan *message.ExpiringMessage, 10)
	s.outputQueue.EXPECT().ResponsesC().
		AnyTimes().Return(outputC)

	s.outputQueue.EXPECT().Start().Times(1)
	s.resolver.EXPECT().Start().Times(1)

	// Expect listener to be reset (i.e. started twice and stopped once)
	s.listener.EXPECT().StartWithContext(gomock.Any()).Times(2)
	s.listener.EXPECT().Stop(gomock.Any()).Times(1)

	s.Require().NoError(s.pipeline.Start())
	s.pipeline.Notify(common.SensorComponentEventCentralReachable)

	outputC <- message.NewExpiring(s.pipeline.context, nil)
	outputC <- message.NewExpiring(s.pipeline.context, nil)

	// Read message A
	msgA, more := <-s.pipeline.ResponsesC()
	s.Require().True(more, "should have more messages in ResponsesC")
	s.Assert().False(msgA.IsExpired(), "context should not be expired")

	s.pipeline.Notify(common.SensorComponentEventOfflineMode)
	s.pipeline.Notify(common.SensorComponentEventCentralReachable)

	// Read message B
	msgB, more := <-s.pipeline.ResponsesC()
	s.Require().True(more, "should have more messages in ResponsesC")
	s.Assert().True(msgB.IsExpired(), "context should be expired")
}

func (s *eventPipelineSuite) Test_ReprocessDeployments() {
	messageReceived := sync.WaitGroup{}
	messageReceived.Add(2)

	msgFromCentral := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ReprocessDeployments{
			ReprocessDeployments: &central.ReprocessDeployments{},
		},
	}
	s.detector.EXPECT().ProcessReprocessDeployments().Times(1).Do(func() {
		defer messageReceived.Done()
	})

	s.resolver.EXPECT().Send(gomock.Any()).Times(1).Do(func(msg interface{}) {
		defer messageReceived.Done()
		resourceEvent, ok := msg.(*component.ResourceEvent)
		assert.True(s.T(), ok)
		assert.NotNil(s.T(), resourceEvent.DeploymentReferences)
		assert.Equal(s.T(), 1, len(resourceEvent.DeploymentReferences))
		assert.NotNil(s.T(), resourceEvent.DeploymentReferences[0].Reference)
		assert.Equal(s.T(), central.ResourceAction_UPDATE_RESOURCE, resourceEvent.DeploymentReferences[0].ParentResourceAction)
		assert.False(s.T(), resourceEvent.DeploymentReferences[0].SkipResolving)
		assert.True(s.T(), resourceEvent.DeploymentReferences[0].ForceDetection)
	})

	err := s.pipeline.ProcessMessage(msgFromCentral)
	s.NoError(err)

	messageReceived.Wait()
}

func (s *eventPipelineSuite) Test_PolicySync() {
	messageReceived := sync.WaitGroup{}
	messageReceived.Add(1)

	msgFromCentral := &central.MsgToSensor{
		Msg: &central.MsgToSensor_PolicySync{
			PolicySync: &central.PolicySync{
				Policies: []*storage.Policy{},
			},
		},
	}
	s.detector.EXPECT().ProcessPolicySync(gomock.Any(), gomock.Any()).Times(1).Do(func(_, _ interface{}) {
		defer messageReceived.Done()
	})

	err := s.pipeline.ProcessMessage(msgFromCentral)
	s.NoError(err)

	messageReceived.Wait()
}

func (s *eventPipelineSuite) Test_ReassessPolicies() {
	messageReceived := sync.WaitGroup{}
	messageReceived.Add(2)

	msgFromCentral := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ReassessPolicies{
			ReassessPolicies: &central.ReassessPolicies{},
		},
	}
	s.detector.EXPECT().ProcessReassessPolicies().Times(1).Do(func() {
		defer messageReceived.Done()
	})

	s.resolver.EXPECT().Send(gomock.Any()).Times(1).Do(func(msg interface{}) {
		defer messageReceived.Done()
		resourceEvent, ok := msg.(*component.ResourceEvent)
		assert.True(s.T(), ok)
		assert.NotNil(s.T(), resourceEvent.DeploymentReferences)
		assert.Equal(s.T(), 1, len(resourceEvent.DeploymentReferences))
		assert.NotNil(s.T(), resourceEvent.DeploymentReferences[0].Reference)
		assert.Equal(s.T(), central.ResourceAction_UPDATE_RESOURCE, resourceEvent.DeploymentReferences[0].ParentResourceAction)
		assert.False(s.T(), resourceEvent.DeploymentReferences[0].SkipResolving)
		assert.True(s.T(), resourceEvent.DeploymentReferences[0].ForceDetection)
	})

	err := s.pipeline.ProcessMessage(msgFromCentral)
	s.NoError(err)

	messageReceived.Wait()
}

func (s *eventPipelineSuite) Test_UpdatedImage() {
	messageReceived := sync.WaitGroup{}
	messageReceived.Add(2)

	msgFromCentral := &central.MsgToSensor{
		Msg: &central.MsgToSensor_UpdatedImage{
			UpdatedImage: &storage.Image{},
		},
	}
	s.detector.EXPECT().ProcessUpdatedImage(gomock.Any()).Times(1).Do(func(msg interface{}) {
		defer messageReceived.Done()
		image, ok := msg.(*storage.Image)
		assert.True(s.T(), ok)
		assert.Equal(s.T(), msgFromCentral.GetUpdatedImage(), image)
	})

	s.resolver.EXPECT().Send(gomock.Any()).Times(1).Do(func(msg interface{}) {
		defer messageReceived.Done()
		resourceEvent, ok := msg.(*component.ResourceEvent)
		assert.True(s.T(), ok)
		assertResourceEvent(s.T(), resourceEvent)
	})

	err := s.pipeline.ProcessMessage(msgFromCentral)
	s.NoError(err)

	messageReceived.Wait()
}

func (s *eventPipelineSuite) Test_ReprocessDeployment() {
	messageReceived := sync.WaitGroup{}
	messageReceived.Add(2)

	msgFromCentral := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ReprocessDeployment{
			ReprocessDeployment: &central.ReprocessDeployment{},
		},
	}
	s.reprocessor.EXPECT().ProcessReprocessDeployments(gomock.Any()).Times(1).Do(func(msg interface{}) {
		defer messageReceived.Done()
		reprocessDeployment, ok := msg.(*central.ReprocessDeployment)
		assert.True(s.T(), ok)
		assert.Equal(s.T(), msgFromCentral.GetReprocessDeployment(), reprocessDeployment)
	})

	s.resolver.EXPECT().Send(gomock.Any()).Times(1).Do(func(msg interface{}) {
		defer messageReceived.Done()
		resourceEvent, ok := msg.(*component.ResourceEvent)
		assert.True(s.T(), ok)
		assertResourceEvent(s.T(), resourceEvent)
	})

	err := s.pipeline.ProcessMessage(msgFromCentral)
	s.NoError(err)

	messageReceived.Wait()
}

func (s *eventPipelineSuite) Test_InvalidateImageCache() {
	messageReceived := sync.WaitGroup{}
	messageReceived.Add(2)

	msgFromCentral := &central.MsgToSensor{
		Msg: &central.MsgToSensor_InvalidateImageCache{
			InvalidateImageCache: &central.InvalidateImageCache{},
		},
	}
	s.reprocessor.EXPECT().ProcessInvalidateImageCache(gomock.Any()).Times(1).Do(func(msg interface{}) {
		defer messageReceived.Done()
		invalidateCache, ok := msg.(*central.InvalidateImageCache)
		assert.True(s.T(), ok)
		assert.Equal(s.T(), msgFromCentral.GetInvalidateImageCache(), invalidateCache)
	})

	s.resolver.EXPECT().Send(gomock.Any()).Times(1).Do(func(msg interface{}) {
		defer messageReceived.Done()
		resourceEvent, ok := msg.(*component.ResourceEvent)
		assert.True(s.T(), ok)
		assertResourceEvent(s.T(), resourceEvent)
	})

	err := s.pipeline.ProcessMessage(msgFromCentral)
	s.NoError(err)

	messageReceived.Wait()
}

func assertResourceEvent(t *testing.T, msg *component.ResourceEvent) {
	assert.NotNil(t, msg.DeploymentReferences)
	assert.Equal(t, 1, len(msg.DeploymentReferences))
	assert.NotNil(t, msg.DeploymentReferences[0].Reference)
	assert.Equal(t, central.ResourceAction_UPDATE_RESOURCE, msg.DeploymentReferences[0].ParentResourceAction)
	assert.True(t, msg.DeploymentReferences[0].SkipResolving)
	assert.True(t, msg.DeploymentReferences[0].ForceDetection)
}
