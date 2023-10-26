package resolver

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/service"
	"github.com/stackrox/rox/sensor/common/store"
	mocksStore "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type resolverSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	mockOutput          *mocks.MockOutputQueue
	mockDeploymentStore *mocksStore.MockDeploymentStore
	mockServiceStore    *mocksStore.MockServiceStore
	mockRBACStore       *mocksStore.MockRBACStore
	mockEndpointManager *mocksStore.MockEndpointManager

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
	s.mockEndpointManager = mocksStore.NewMockEndpointManager(s.mockCtrl)

	s.resolver = New(s.mockOutput, &fakeProvider{
		deploymentStore: s.mockDeploymentStore,
		serviceStore:    s.mockServiceStore,
		rbacStore:       s.mockRBACStore,
		endpointManager: s.mockEndpointManager,
	}, 100)
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
		deploymentID    string
		permissionLevel storage.PermissionLevel
	}{
		"[1234]: None": {
			deploymentID:    "1234",
			permissionLevel: storage.PermissionLevel_NONE,
		},
		"[1234]: Elevated in namespace": {
			deploymentID:    "1234",
			permissionLevel: storage.PermissionLevel_ELEVATED_IN_NAMESPACE,
		},
		"[4321]: Elevated in cluster": {
			deploymentID:    "4321",
			permissionLevel: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE,
		},
	}

	for name, testCase := range testCases {
		s.Run(name, func() {
			messageReceived := sync.WaitGroup{}
			messageReceived.Add(1)

			s.givenPermissionLevelForDeployment(testCase.deploymentID, testCase.permissionLevel)

			expectedDeployment := deploymentMatcher{
				id:                    testCase.deploymentID,
				permissionLevel:       testCase.permissionLevel,
				expectedExposureInfos: nil,
			}

			s.mockOutput.EXPECT().Send(&expectedDeployment).Times(1).Do(func(arg0 interface{}) {
				defer messageReceived.Done()
			})

			s.resolver.Send(&component.ResourceEvent{
				DeploymentReferences: []component.DeploymentReference{
					{
						Reference:            resolver.ResolveDeploymentIds(testCase.deploymentID),
						ParentResourceAction: central.ResourceAction_UPDATE_RESOURCE,
					},
				},
			})

			messageReceived.Wait()
		})
	}
}

func (s *resolverSuite) Test_Send_DeploymentsWithServiceExposure() {
	err := s.resolver.Start()
	s.NoError(err)

	messageReceived := sync.WaitGroup{}
	messageReceived.Add(1)

	s.givenServiceExposureForDeployment("1234", []map[service.PortRef][]*storage.PortConfig_ExposureInfo{
		s.givenStubPortExposure(),
	})

	expectedDeployment := deploymentMatcher{
		id:              "1234",
		permissionLevel: storage.PermissionLevel_NONE,
		expectedExposureInfos: []*storage.PortConfig_ExposureInfo{
			{
				Level:       storage.PortConfig_EXTERNAL,
				ServiceName: "my.service",
				ServicePort: 80,
			},
		},
	}

	s.mockOutput.EXPECT().Send(&expectedDeployment).Times(1).Do(func(arg0 interface{}) {
		defer messageReceived.Done()
	})

	s.resolver.Send(&component.ResourceEvent{
		DeploymentReferences: []component.DeploymentReference{
			{
				Reference:            resolver.ResolveDeploymentIds("1234"),
				ParentResourceAction: central.ResourceAction_UPDATE_RESOURCE,
			},
		},
	})

	messageReceived.Wait()
}

func (s *resolverSuite) Test_Send_MultipleDeploymentRefs() {
	err := s.resolver.Start()
	s.NoError(err)

	messageReceived := sync.WaitGroup{}
	messageReceived.Add(1)

	s.givenPermissionLevelForDeployment("1234", storage.PermissionLevel_NONE)
	s.givenPermissionLevelForDeployment("4321", storage.PermissionLevel_ELEVATED_IN_NAMESPACE)
	s.givenPermissionLevelForDeployment("6543", storage.PermissionLevel_ELEVATED_CLUSTER_WIDE)

	s.mockOutput.EXPECT().Send(&messageCounterMatcher{numEvents: 3}).Times(1).Do(func(arg0 interface{}) {
		defer messageReceived.Done()
	})

	s.resolver.Send(&component.ResourceEvent{
		DeploymentReferences: []component.DeploymentReference{
			{
				Reference:            resolver.ResolveDeploymentIds("1234", "4321"),
				ParentResourceAction: central.ResourceAction_UPDATE_RESOURCE,
			},
			{
				Reference:            resolver.ResolveDeploymentIds("6543"),
				ParentResourceAction: central.ResourceAction_UPDATE_RESOURCE,
			},
		},
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
				DeploymentReferences: []component.DeploymentReference{
					{
						Reference:            resolver.ResolveDeploymentIds("1234"),
						ParentResourceAction: action,
					},
				},
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
		DeploymentReferences: []component.DeploymentReference{
			{
				Reference:            resolver.ResolveDeploymentIds("1234"),
				ParentResourceAction: central.ResourceAction_UPDATE_RESOURCE,
			},
		},
	})

	messageReceived.Wait()
}

func (s *resolverSuite) Test_Send_DeploymentNotFound() {
	err := s.resolver.Start()
	s.NoError(err)

	messageReceived := sync.WaitGroup{}
	messageReceived.Add(1)

	s.givenNilDeployment()

	s.mockEndpointManager.EXPECT().OnDeploymentCreateOrUpdateByID(gomock.Any()).Times(0)
	s.mockRBACStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).Times(0)
	s.mockDeploymentStore.EXPECT().BuildDeploymentWithDependencies(gomock.Any(), gomock.Any()).Times(0)

	s.mockOutput.EXPECT().Send(&messageCounterMatcher{numEvents: 0}).Times(1).Do(func(arg0 interface{}) {
		defer messageReceived.Done()
	})

	s.resolver.Send(&component.ResourceEvent{
		DeploymentReferences: []component.DeploymentReference{
			{
				Reference:            resolver.ResolveDeploymentIds("1234"),
				ParentResourceAction: central.ResourceAction_UPDATE_RESOURCE,
			},
		},
	})

	messageReceived.Wait()
}

func (s *resolverSuite) Test_Send_DetectorReference() {
	err := s.resolver.Start()
	s.NoError(err)

	messageReceived := sync.WaitGroup{}
	messageReceived.Add(1)

	detectionObject := []component.DetectorMessage{
		{
			Object: &storage.Deployment{Id: "1234"},
			Action: central.ResourceAction_UPDATE_RESOURCE,
		},
	}

	s.mockOutput.EXPECT().Send(&detectionObjectMatcher{expected: detectionObject}).Times(1).Do(func(arg0 interface{}) {
		defer messageReceived.Done()
	})

	s.resolver.Send(&component.ResourceEvent{
		DetectorMessages: detectionObject,
	})

	messageReceived.Wait()
}

func (s *resolverSuite) Test_Send_ForwardedMessagesAreSent() {
	err := s.resolver.Start()
	s.NoError(err)

	// There are two types of resource events that will be written to the output queue.
	// 1) Resource events that were processed at the handlers level. E.g.: Pod events,
	// these will be passed to the resolver component through the `ForwardMessages` property
	// in `component.ResourceEvent`.
	// 2) Deployments that need to be processed against their dependencies. These come
	// as deployment references, then the resource event is generated in this component.
	// All events are merged in the same `ForwardedMessages` in the end, and passed to the
	// output component to be sent to central.
	testCases := map[string]struct {
		resolver                    resolver.DeploymentReference
		forwardedMessages           []*central.SensorEvent
		expectedDeploymentProcessed int
		expectedEvents              int
	}{
		"Single id, no forwarded messages": {
			resolver:                    resolver.ResolveDeploymentIds("1234"),
			forwardedMessages:           nil,
			expectedDeploymentProcessed: 1,
			expectedEvents:              1,
		},
		"Multiple ids, no forwarded messages": {
			resolver:                    resolver.ResolveDeploymentIds("1234", "4321"),
			forwardedMessages:           nil,
			expectedDeploymentProcessed: 2,
			expectedEvents:              2,
		},
		"Single id, one forwarded message": {
			resolver:                    resolver.ResolveDeploymentIds("1234"),
			forwardedMessages:           []*central.SensorEvent{s.givenStubSensorEvent()},
			expectedDeploymentProcessed: 1,
			expectedEvents:              2,
		},
		"Single id, multiple forwarded messages": {
			resolver:                    resolver.ResolveDeploymentIds("1234"),
			forwardedMessages:           []*central.SensorEvent{s.givenStubSensorEvent(), s.givenStubSensorEvent()},
			expectedDeploymentProcessed: 1,
			expectedEvents:              3,
		},
		"No deployment resolver, multiple forwarded messages": {
			resolver:                    nil,
			forwardedMessages:           []*central.SensorEvent{s.givenStubSensorEvent(), s.givenStubSensorEvent()},
			expectedDeploymentProcessed: 0,
			expectedEvents:              2,
		},
	}

	for name, testCase := range testCases {
		s.Run(name, func() {
			messageReceived := sync.WaitGroup{}
			messageReceived.Add(1)

			s.givenAnyDeploymentProcessedNTimes(testCase.expectedDeploymentProcessed)

			s.mockOutput.EXPECT().Send(&messageCounterMatcher{numEvents: testCase.expectedEvents}).Times(1).Do(func(arg0 interface{}) {
				defer messageReceived.Done()
			})

			s.resolver.Send(&component.ResourceEvent{
				ForwardMessages: testCase.forwardedMessages,
				DeploymentReferences: []component.DeploymentReference{
					{
						Reference:            testCase.resolver,
						ParentResourceAction: central.ResourceAction_UPDATE_RESOURCE,
					},
				},
			})

			messageReceived.Wait()
		})
	}
}

func (s *resolverSuite) givenStubSensorEvent() *central.SensorEvent {
	return new(central.SensorEvent)
}

func (s *resolverSuite) givenStubPortExposure() map[service.PortRef][]*storage.PortConfig_ExposureInfo {
	return map[service.PortRef][]*storage.PortConfig_ExposureInfo{
		{
			Port:     intstr.IntOrString{IntVal: 8080},
			Protocol: "TCP",
		}: {
			{
				Level:       storage.PortConfig_EXTERNAL,
				ServiceName: "my.service",
				ServicePort: 80,
			},
		},
	}
}

func (s *resolverSuite) givenBuildDependenciesError(deployment string) {
	s.mockDeploymentStore.EXPECT().Get(gomock.Eq(deployment)).Times(1).DoAndReturn(func(arg0 interface{}) *storage.Deployment {
		return &storage.Deployment{}
	})
	s.mockEndpointManager.EXPECT().OnDeploymentCreateOrUpdateByID(gomock.Eq(deployment)).Times(1)
	s.mockRBACStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).Times(1).
		DoAndReturn(func(arg0 interface{}) storage.PermissionLevel { return storage.PermissionLevel_NONE })
	s.mockServiceStore.EXPECT().GetExposureInfos(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(arg0, arg1 interface{}) []map[service.PortRef][]*storage.PortConfig_ExposureInfo { return nil })

	s.mockDeploymentStore.EXPECT().BuildDeploymentWithDependencies(
		gomock.Eq(deployment), gomock.Eq(store.Dependencies{
			PermissionLevel: storage.PermissionLevel_NONE,
			Exposures:       nil,
			LocalImages:     set.NewStringSet(),
		})).
		Times(1).
		DoAndReturn(func(arg0, arg1 interface{}) (*storage.Deployment, bool, error) {
			return nil, false, errors.New("dependency error")
		})
}

func (s *resolverSuite) givenNilDeployment() {
	s.mockDeploymentStore.EXPECT().Get(gomock.Any()).Times(1).DoAndReturn(func(arg0 interface{}) *storage.Deployment {
		return nil
	})
}

func (s *resolverSuite) givenPermissionLevelForDeployment(deployment string, permissionLevel storage.PermissionLevel) {
	s.mockDeploymentStore.EXPECT().Get(gomock.Eq(deployment)).Times(1).DoAndReturn(func(arg0 interface{}) *storage.Deployment {
		return &storage.Deployment{
			Labels: map[string]string{},
		}
	})

	s.mockEndpointManager.EXPECT().OnDeploymentCreateOrUpdateByID(gomock.Eq(deployment)).Times(1)

	s.mockServiceStore.EXPECT().GetExposureInfos(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(arg0, arg1 interface{}) []map[service.PortRef][]*storage.PortConfig_ExposureInfo {
		return nil
	})

	s.mockRBACStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).Times(1).
		DoAndReturn(func(arg0 interface{}) storage.PermissionLevel { return permissionLevel })

	s.mockDeploymentStore.EXPECT().BuildDeploymentWithDependencies(
		gomock.Eq(deployment), gomock.Eq(store.Dependencies{
			PermissionLevel: permissionLevel,
			Exposures:       nil,
			LocalImages:     set.NewStringSet(),
		})).
		Times(1).
		DoAndReturn(func(arg0, arg1 interface{}) (*storage.Deployment, bool, error) {
			return &storage.Deployment{Id: deployment, ServiceAccountPermissionLevel: permissionLevel}, true, nil
		})
}

func (s *resolverSuite) givenServiceExposureForDeployment(deployment string, exposure []map[service.PortRef][]*storage.PortConfig_ExposureInfo) {
	s.mockDeploymentStore.EXPECT().Get(gomock.Eq(deployment)).Times(1).DoAndReturn(func(arg0 interface{}) *storage.Deployment {
		return &storage.Deployment{
			Namespace: "example",
			Labels:    map[string]string{"app": "a"},
		}
	})

	s.mockEndpointManager.EXPECT().OnDeploymentCreateOrUpdateByID(gomock.Eq(deployment)).Times(1)

	s.mockRBACStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).AnyTimes().
		DoAndReturn(func(arg0 interface{}) storage.PermissionLevel { return storage.PermissionLevel_NONE })

	s.mockServiceStore.EXPECT().GetExposureInfos(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(arg0, arg1 interface{}) []map[service.PortRef][]*storage.PortConfig_ExposureInfo { return exposure })

	var flatExposures []*storage.PortConfig_ExposureInfo
	for _, e := range exposure {
		for _, list := range e {
			flatExposures = append(flatExposures, list...)
		}
	}

	s.mockDeploymentStore.EXPECT().BuildDeploymentWithDependencies(
		gomock.Eq(deployment), gomock.Eq(store.Dependencies{
			PermissionLevel: storage.PermissionLevel_NONE,
			Exposures:       exposure,
			LocalImages:     set.NewStringSet(),
		})).
		Times(1).
		DoAndReturn(func(arg0, arg1 interface{}) (*storage.Deployment, bool, error) {
			return &storage.Deployment{
				Id:                            deployment,
				ServiceAccountPermissionLevel: storage.PermissionLevel_NONE,
				Ports: []*storage.PortConfig{
					{
						ExposureInfos: flatExposures,
					},
				},
			}, true, nil
		})
}

func (s *resolverSuite) givenAnyDeploymentProcessedNTimes(times int) {
	s.mockDeploymentStore.EXPECT().Get(gomock.Any()).Times(times).DoAndReturn(func(arg0 interface{}) *storage.Deployment {
		return &storage.Deployment{}
	})

	s.mockEndpointManager.EXPECT().OnDeploymentCreateOrUpdateByID(gomock.Any()).Times(times)

	s.mockRBACStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).Times(times).
		DoAndReturn(func(arg0 interface{}) storage.PermissionLevel { return storage.PermissionLevel_DEFAULT })

	s.mockServiceStore.EXPECT().GetExposureInfos(gomock.Any(), gomock.Any()).Times(times).
		DoAndReturn(func(arg0, arg1 interface{}) []map[service.PortRef][]*storage.PortConfig_ExposureInfo { return nil })

	s.mockDeploymentStore.EXPECT().BuildDeploymentWithDependencies(gomock.Any(), gomock.Any()).
		Times(times).
		DoAndReturn(func(arg0, arg1 interface{}) (*storage.Deployment, bool, error) {
			return &storage.Deployment{}, true, nil
		})
}

type deploymentMatcher struct {
	id                    string
	permissionLevel       storage.PermissionLevel
	expectedExposureInfos []*storage.PortConfig_ExposureInfo
	error                 string
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

	if m.expectedExposureInfos != nil && len(m.expectedExposureInfos) > 0 {
		if len(deployment.GetPorts()) == 0 {
			m.error = fmt.Sprintf("No ports on deployment object: %v", deployment)
			return false
		}

		if !cmp.Equal(m.expectedExposureInfos, deployment.GetPorts()[0].GetExposureInfos()) {
			diff := cmp.Diff(m.expectedExposureInfos, deployment.GetPorts()[0].GetExposureInfos())
			m.error = fmt.Sprintf("Exposure info differs: %s", diff)
			return false
		}
	}

	return true
}

func (m *deploymentMatcher) String() string {
	return fmt.Sprintf("Deployment (%s) (Permission: %s): %s", m.id, m.permissionLevel, m.error)
}

type detectionObjectMatcher struct {
	expected []component.DetectorMessage
	error    string
}

func (m *detectionObjectMatcher) Matches(target interface{}) bool {
	event, ok := target.(*component.ResourceEvent)
	if !ok {
		m.error = "received message isn't a resource event"
		return false
	}

	if !cmp.Equal(m.expected, event.DetectorMessages) {
		m.error = fmt.Sprintf("received detection deployment doesn't match expected: %s", cmp.Diff(m.expected, event.ReprocessDeployments))
		return false
	}

	return true
}

func (m *detectionObjectMatcher) String() string {
	return fmt.Sprintf("expected %v: error %s", m.expected, m.error)
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
	deploymentStore *mocksStore.MockDeploymentStore
	serviceStore    *mocksStore.MockServiceStore
	rbacStore       *mocksStore.MockRBACStore
	endpointManager *mocksStore.MockEndpointManager
}

func (p *fakeProvider) Deployments() store.DeploymentStore {
	return p.deploymentStore
}

func (p *fakeProvider) Services() store.ServiceStore {
	return p.serviceStore
}

func (p *fakeProvider) RBAC() store.RBACStore {
	return p.rbacStore
}

func (p *fakeProvider) EndpointManager() store.EndpointManager {
	return p.endpointManager
}

func (p *fakeProvider) Pods() store.PodStore {
	return nil
}

func (p *fakeProvider) Registries() *registry.Store {
	return nil
}

func (p *fakeProvider) ServiceAccounts() store.ServiceAccountStore {
	return nil
}

func (p *fakeProvider) NetworkPolicies() store.NetworkPolicyStore {
	return nil
}

func (p *fakeProvider) Entities() *clusterentities.Store {
	return nil
}
