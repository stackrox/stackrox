package resolver

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component/mocks"
	"github.com/stretchr/testify/suite"
)

type resolverSuite struct {
	suite.Suite

	mockOutput *mocks.MockOutputQueue
}

var _ suite.SetupTestSuite = &resolverSuite{}

func TestResolverSuite(t *testing.T) {
	suite.Run(t, new(resolverSuite))
}

func (s *resolverSuite) SetupTest() {
	mockCtrl := gomock.NewController(s.T())
	s.mockOutput = mocks.NewMockOutputQueue(mockCtrl)
}

func (s *resolverSuite) Test_InitializeResolver() {
	resolver := New(s.mockOutput)
	err := resolver.Start()
	s.NoError(err)
}

func (s *resolverSuite) Test_MessageSentToOutput() {
	resolver := New(s.mockOutput)
	err := resolver.Start()
	s.NoError(err)

	messageReceived := sync.WaitGroup{}
	messageReceived.Add(1)

	s.mockOutput.EXPECT().Send(gomock.Any()).Times(1).Do(func(arg0 interface{}) {
		defer messageReceived.Done()
	})

	resolver.Send(&component.ResourceEvent{
		ForwardMessages: []*central.SensorEvent{
			{
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Resource: &central.SensorEvent_Deployment{
					Deployment: &storage.Deployment{Id: "abc"},
				},
			},
		},
	})

	messageReceived.Wait()
}
