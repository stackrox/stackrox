package eventpipeline

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
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
	pipeline    *eventPipeline
}

var _ suite.SetupTestSuite = &eventPipelineSuite{}
var _ suite.TearDownTestSuite = &eventPipelineSuite{}

func TestEventPipelineSuite(t *testing.T) {
	suite.Run(t, new(eventPipelineSuite))
}

func (s *eventPipelineSuite) TearDownTest() {
	s.T().Cleanup(s.mockCtrl.Finish)
}

type mockListener struct{}

func (m *mockListener) Start() error { return nil }
func (m *mockListener) Stop(_ error) {}

func (s *eventPipelineSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.resolver = mockComponent.NewMockResolver(s.mockCtrl)
	s.detector = mockDetector.NewMockDetector(s.mockCtrl)
	s.reprocessor = mockReprocessor.NewMockHandler(s.mockCtrl)
	s.pipeline = &eventPipeline{
		eventsC:     make(chan *message.ExpiringMessage),
		stopSig:     concurrency.NewSignal(),
		output:      mockComponent.NewMockOutputQueue(s.mockCtrl),
		resolver:    s.resolver,
		detector:    s.detector,
		reprocessor: s.reprocessor,
		listener:    &mockListener{},
	}
}

func (s *eventPipelineSuite) Test_ReprocessDeployments() {
	s.T().Setenv("ROX_RESYNC_DISABLED", "true")
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
	s.T().Setenv("ROX_RESYNC_DISABLED", "true")
	messageReceived := sync.WaitGroup{}
	messageReceived.Add(1)

	msgFromCentral := &central.MsgToSensor{
		Msg: &central.MsgToSensor_PolicySync{
			PolicySync: &central.PolicySync{
				Policies: []*storage.Policy{},
			},
		},
	}
	s.detector.EXPECT().ProcessPolicySync(gomock.Any()).Times(1).Do(func(_ interface{}) {
		defer messageReceived.Done()
	})

	err := s.pipeline.ProcessMessage(msgFromCentral)
	s.NoError(err)

	messageReceived.Wait()
}

func (s *eventPipelineSuite) Test_ReassessPolicies() {
	s.T().Setenv("ROX_RESYNC_DISABLED", "true")
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
	s.T().Setenv("ROX_RESYNC_DISABLED", "true")
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
	s.T().Setenv("ROX_RESYNC_DISABLED", "true")
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
	s.T().Setenv("ROX_RESYNC_DISABLED", "true")
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
