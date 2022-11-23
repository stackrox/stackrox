package resolver

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/store"
	mocksStore "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component/mocks"
	"github.com/stretchr/testify/suite"
)

type resolverSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	mockOutput          *mocks.MockOutputQueue
	mockDeploymentStore *mocksStore.MockDeploymentStore
	mockServiceStore    *mocksStore.MockServiceStore
	mockRBACStore       *mocksStore.MockRBACStore

	resolver component.Resolver
}

var _ suite.SetupTestSuite = &resolverSuite{}
var _ suite.TearDownTestSuite = &resolverSuite{}

func TestResolverSuite(t *testing.T) {
	suite.Run(t, new(resolverSuite))
}

func (s *resolverSuite) TearDownTest() {
	s.T().Cleanup(s.mockCtrl.Finish)
}

func (s *resolverSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.mockOutput = mocks.NewMockOutputQueue(s.mockCtrl)
	s.mockDeploymentStore = mocksStore.NewMockDeploymentStore(s.mockCtrl)
	s.mockServiceStore = mocksStore.NewMockServiceStore(s.mockCtrl)
	s.mockRBACStore = mocksStore.NewMockRBACStore(s.mockCtrl)

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

func (s *resolverSuite) Test_Send_DeploymentWithRBACs() {
	err := s.resolver.Start()
	s.NoError(err)

	testCases := map[string]struct {
		deploymentId    string
		permissionLevel storage.PermissionLevel
	}{
		"[1234]: None": {
			deploymentId:    "1234",
			permissionLevel: storage.PermissionLevel_NONE,
		},
		"[1234]: Elevated in namespace": {
			deploymentId:    "1234",
			permissionLevel: storage.PermissionLevel_ELEVATED_IN_NAMESPACE,
		},
		"[4321]: Elevated in cluster": {
			deploymentId:    "4321",
			permissionLevel: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE,
		},
	}

	for name, testCase := range testCases {
		s.Run(name, func() {
			messageReceived := sync.WaitGroup{}
			messageReceived.Add(1)

			s.givenPermissionLevelForDeployment(testCase.deploymentId, testCase.permissionLevel)

			expectedDeployment := deploymentMatcher{
				id:              testCase.deploymentId,
				permissionLevel: testCase.permissionLevel,
				exposure:        nil,
			}

			s.mockOutput.EXPECT().Send(&expectedDeployment).Times(1).Do(func(arg0 interface{}) {
				defer messageReceived.Done()
			})

			s.resolver.Send(&component.ResourceEvent{
				DeploymentReference: resolver.ResolveDeploymentIds(testCase.deploymentId),
			})

			messageReceived.Wait()
		})
	}
}

func (s *resolverSuite) Test_Send_MultipleDeploymentRefs() {
	err := s.resolver.Start()
	s.NoError(err)

	messageReceived := sync.WaitGroup{}
	messageReceived.Add(1)

	s.givenPermissionLevelForDeployment("1234", storage.PermissionLevel_NONE)
	s.givenPermissionLevelForDeployment("4321", storage.PermissionLevel_ELEVATED_IN_NAMESPACE)

	s.mockOutput.EXPECT().Send(&messageCounterMatcher{numEvents: 2}).Times(1).Do(func(arg0 interface{}) {
		defer messageReceived.Done()
	})

	s.resolver.Send(&component.ResourceEvent{
		DeploymentReference: resolver.ResolveDeploymentIds("1234", "4321"),
	})

	messageReceived.Wait()
}

func (s *resolverSuite) Test_Send_ResourceAction() {
	err := s.resolver.Start()
	s.NoError(err)

	for _, action := range []central.ResourceAction{central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE} {
		s.Run(fmt.Sprintf("ResourceAction: %s", action), func() {
			messageReceived := sync.WaitGroup{}
			messageReceived.Add(1)

			s.givenPermissionLevelForDeployment("1234", storage.PermissionLevel_NONE)
			s.mockOutput.EXPECT()

			s.mockOutput.EXPECT().Send(&resourceActionMatcher{resourceAction: action}).Times(1).Do(func(arg0 interface{}) {
				defer messageReceived.Done()
			})

			s.resolver.Send(&component.ResourceEvent{
				DeploymentReference:  resolver.ResolveDeploymentIds("1234"),
				ParentResourceAction: action,
			})

			messageReceived.Wait()
		})
	}
}

func (s *resolverSuite) Test_Send_BuildDeploymentWithDependenciesError() {
	err := s.resolver.Start()
	s.NoError(err)

	messageReceived := sync.WaitGroup{}
	messageReceived.Add(1)

	s.givenBuildDependenciesError("1234")

	s.mockOutput.EXPECT().Send(&messageCounterMatcher{numEvents: 0}).Times(1).Do(func(arg0 interface{}) {
		defer messageReceived.Done()
	})

	s.resolver.Send(&component.ResourceEvent{
		DeploymentReference: resolver.ResolveDeploymentIds("1234"),
	})

	messageReceived.Wait()
}

func (s *resolverSuite) Test_Send_DeploymentNotFound() {
	err := s.resolver.Start()
	s.NoError(err)

	messageReceived := sync.WaitGroup{}
	messageReceived.Add(1)

	s.givenNilDeploymentForId()

	s.mockRBACStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).Times(0)
	s.mockDeploymentStore.EXPECT().BuildDeploymentWithDependencies(gomock.Any(), gomock.Any()).Times(0)

	s.mockOutput.EXPECT().Send(&messageCounterMatcher{numEvents: 0}).Times(1).Do(func(arg0 interface{}) {
		defer messageReceived.Done()
	})

	s.resolver.Send(&component.ResourceEvent{
		DeploymentReference: resolver.ResolveDeploymentIds("1234"),
	})

	messageReceived.Wait()
}

func (s *resolverSuite) givenBuildDependenciesError(deployment string) {
	s.mockDeploymentStore.EXPECT().Get(gomock.Eq(deployment)).Times(1).DoAndReturn(func(arg0 interface{}) *storage.Deployment {
		return &storage.Deployment{}
	})
	s.mockRBACStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).Times(1).
		DoAndReturn(func(arg0 interface{}) storage.PermissionLevel { return storage.PermissionLevel_NONE })

	s.mockDeploymentStore.EXPECT().BuildDeploymentWithDependencies(
		gomock.Eq(deployment), gomock.Eq(store.Dependencies{
			PermissionLevel: storage.PermissionLevel_NONE,
			Exposures:       nil,
		})).
		Times(1).
		DoAndReturn(func(arg0, arg1 interface{}) (*storage.Deployment, error) {
			return nil, errors.New("dependency error")
		})
}

func (s *resolverSuite) givenNilDeploymentForId() {
	s.mockDeploymentStore.EXPECT().Get(gomock.Any()).Times(1).DoAndReturn(func(arg0 interface{}) *storage.Deployment {
		return nil
	})
}

func (s *resolverSuite) givenPermissionLevelForDeployment(deployment string, permissionLevel storage.PermissionLevel) {
	s.mockDeploymentStore.EXPECT().Get(gomock.Eq(deployment)).Times(1).DoAndReturn(func(arg0 interface{}) *storage.Deployment {
		return &storage.Deployment{}
	})
	s.mockRBACStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).Times(1).
		DoAndReturn(func(arg0 interface{}) storage.PermissionLevel { return permissionLevel })

	s.mockDeploymentStore.EXPECT().BuildDeploymentWithDependencies(
		gomock.Eq(deployment), gomock.Eq(store.Dependencies{
			PermissionLevel: permissionLevel,
			Exposures:       nil,
		})).
		Times(1).
		DoAndReturn(func(arg0, arg1 interface{}) (*storage.Deployment, error) {
			return &storage.Deployment{Id: deployment, ServiceAccountPermissionLevel: permissionLevel}, nil
		})
}

type deploymentMatcher struct {
	id              string
	permissionLevel storage.PermissionLevel
	exposure        interface{}
	error           string
}

func (m *deploymentMatcher) Matches(target interface{}) bool {
	event, ok := target.(*component.ResourceEvent)
	if !ok {
		m.error = "received message isn't a resource event"
		return false
	}

	if len(event.ForwardMessages) < 1 {
		m.error = fmt.Sprintf("not enough ForwardMessages: %d", len(event.ForwardMessages))
		return false
	}

	deployment := event.ForwardMessages[0].GetDeployment()
	if deployment == nil {
		m.error = "no deployment in resource event message"
		return false
	}

	if deployment.GetId() != m.id {
		m.error = fmt.Sprintf("IDs don't match: expected %s != %s", m.id, deployment.GetId())
		return false
	}

	if deployment.GetServiceAccountPermissionLevel() != m.permissionLevel {
		m.error = fmt.Sprintf("Permission level doesn't match %s != %s", m.permissionLevel, deployment.GetServiceAccountPermissionLevel())
		return false
	}

	return true
}

func (m *deploymentMatcher) String() string {
	return fmt.Sprintf("Deployment (%s) (Permission: %s): %s", m.id, m.permissionLevel, m.error)
}

type messageCounterMatcher struct {
	numEvents int
	error     string
}

func (m *messageCounterMatcher) Matches(target interface{}) bool {
	event, ok := target.(*component.ResourceEvent)
	if !ok {
		m.error = "received message isn't a resource event"
		return false
	}

	if len(event.ForwardMessages) != m.numEvents {
		m.error = fmt.Sprintf("expected %d events but received %d", m.numEvents, len(event.ForwardMessages))
		return false
	}

	return true
}

func (m *messageCounterMatcher) String() string {
	return fmt.Sprintf("expected %d: error %s", m.numEvents, m.error)
}

type resourceActionMatcher struct {
	resourceAction central.ResourceAction
	error          string
}

func (m *resourceActionMatcher) Matches(target interface{}) bool {
	event, ok := target.(*component.ResourceEvent)
	if !ok {
		m.error = "received message isn't a resource event"
		return false
	}

	if len(event.ForwardMessages) < 1 {
		m.error = fmt.Sprintf("not enough ForwardMessages: %d", len(event.ForwardMessages))
		return false
	}

	if event.ForwardMessages[0].GetAction() != m.resourceAction {
		m.error = fmt.Sprintf("expected %s action but received %s", m.resourceAction, event.ForwardMessages[0].GetAction())
		return false
	}

	return true
}

func (m *resourceActionMatcher) String() string {
	return fmt.Sprintf("expected %d: error %s", m.resourceAction, m.error)
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
