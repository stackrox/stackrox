package resolver

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/store"
	mocksStore "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component/mocks"
	"github.com/stretchr/testify/suite"
)

type resolverSuite struct {
	suite.Suite

	mockOutput          *mocks.MockOutputQueue
	mockDeploymentStore *mocksStore.MockDeploymentStore
	mockServiceStore    *mocksStore.MockServiceStore
	mockRBACStore       *mocksStore.MockRBACStore

	resolver component.Resolver
}

var _ suite.SetupTestSuite = &resolverSuite{}

func TestResolverSuite(t *testing.T) {
	suite.Run(t, new(resolverSuite))
}

func (s *resolverSuite) SetupTest() {
	mockCtrl := gomock.NewController(s.T())

	s.mockOutput = mocks.NewMockOutputQueue(mockCtrl)
	s.mockDeploymentStore = mocksStore.NewMockDeploymentStore(mockCtrl)
	s.mockServiceStore = mocksStore.NewMockServiceStore(mockCtrl)
	s.mockRBACStore = mocksStore.NewMockRBACStore(mockCtrl)

	s.resolver = New(s.mockOutput, s.mockDeploymentStore, &fakeProvider{
		serviceStore: s.mockServiceStore,
		rbacStore:    s.mockRBACStore,
	})
}

func (s *resolverSuite) Test_InitializeResolver() {
	err := s.resolver.Start()
	s.NoError(err)
}

func (s *resolverSuite) Test_MessageSentToOutput() {
	err := s.resolver.Start()
	s.NoError(err)

	messageReceived := sync.WaitGroup{}
	messageReceived.Add(1)

	s.mockOutput.EXPECT().Send(gomock.Any()).Times(1).Do(func(arg0 interface{}) {
		defer messageReceived.Done()
	})

	s.resolver.Send(&component.ResourceEvent{
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

type fakeProvider struct {
	serviceStore *mocksStore.MockServiceStore
	rbacStore    *mocksStore.MockRBACStore
}

func (p *fakeProvider) Services() store.ServiceStore {
	return p.serviceStore
}

func (p *fakeProvider) RBAC() store.RBACStore {
	return p.rbacStore
}
