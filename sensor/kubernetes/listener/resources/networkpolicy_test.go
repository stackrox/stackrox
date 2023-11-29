package resources

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/sensor/common/detector/mocks"
	mocksStore "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	networkingV1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestNetworkPolicyDispatcher(t *testing.T) {
	suite.Run(t, new(NetworkPolicyDispatcherSuite))
}

type NetworkPolicyDispatcherSuite struct {
	suite.Suite

	mockCtrl        *gomock.Controller
	netpolStore     *mocksStore.MockNetworkPolicyStore
	deploymentStore *DeploymentStore
	detector        *mocks.MockDetector
	dispatcher      *networkPolicyDispatcher
}

var _ suite.SetupTestSuite = (*NetworkPolicyDispatcherSuite)(nil)
var _ suite.TearDownTestSuite = (*NetworkPolicyDispatcherSuite)(nil)

func (suite *NetworkPolicyDispatcherSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.netpolStore = mocksStore.NewMockNetworkPolicyStore(suite.mockCtrl)
	suite.deploymentStore = newDeploymentStore()
	suite.detector = mocks.NewMockDetector(suite.mockCtrl)

	suite.dispatcher = newNetworkPolicyDispatcher(suite.netpolStore, suite.deploymentStore)

	// TODO(ROX-9990): Use the DeploymentStore mock
	deployments := []*deploymentWrap{
		{
			Deployment: &storage.Deployment{
				Name:      "deploy-1",
				Id:        "1",
				Namespace: "default",
				PodLabels: map[string]string{
					"app":  "sensor",
					"role": "backend",
				},
			},
		},
		{
			Deployment: &storage.Deployment{
				Name:      "deploy-2",
				Id:        "2",
				Namespace: "default",
				PodLabels: map[string]string{},
			},
		},
		{
			Deployment: &storage.Deployment{
				Name:      "deploy-3",
				Id:        "3",
				Namespace: "secure",
				PodLabels: map[string]string{
					"app":  "sensor",
					"role": "backend",
				},
			},
		},
		{
			Deployment: &storage.Deployment{
				Name:      "deploy-4",
				Id:        "4",
				Namespace: "default",
				PodLabels: map[string]string{
					"app": "sensor-2",
				},
			},
		},
	}

	for _, d := range deployments {
		suite.deploymentStore.addOrUpdateDeployment(d)
	}
}

func (suite *NetworkPolicyDispatcherSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func createNetworkPolicy(id, namespace string, podSelector map[string]string) *networkingV1.NetworkPolicy {
	netpol := &networkingV1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID(id),
			Name:      "network-policy",
			Namespace: namespace,
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
		},
		Spec: networkingV1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			Ingress:     nil,
			Egress:      nil,
		},
	}
	protoconv.ConvertTimeToTimestamp(netpol.GetCreationTimestamp().Time)
	if len(podSelector) > 0 {
		netpol.Spec.PodSelector.MatchLabels = podSelector
	}
	return netpol
}

func createSensorEvent(np *networkingV1.NetworkPolicy, action central.ResourceAction) map[string]*central.SensorEvent {
	return map[string]*central.SensorEvent{
		string(np.UID): {
			Id:     string(np.UID),
			Action: action,
			Resource: &central.SensorEvent_NetworkPolicy{
				NetworkPolicy: networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: np}.ToRoxNetworkPolicy(),
			},
		},
	}
}

func (suite *NetworkPolicyDispatcherSuite) Test_ProcessEvent() {
	cases := map[string]struct {
		netpol              interface{}
		oldNetpol           interface{}
		action              central.ResourceAction
		expectedEvents      map[string]*central.SensorEvent
		expectedDeployments []*deploymentWrap
	}{
		"New NetworkPolicy": {
			netpol: createNetworkPolicy("1", "default", map[string]string{
				"app": "sensor",
			}),
			oldNetpol:      nil,
			action:         central.ResourceAction_CREATE_RESOURCE,
			expectedEvents: nil,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
			},
		},
		"New NetworkPolicy, no selector": {
			netpol:         createNetworkPolicy("1", "default", nil),
			oldNetpol:      nil,
			action:         central.ResourceAction_CREATE_RESOURCE,
			expectedEvents: nil,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "2",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "4",
						Namespace: "default",
					},
				},
			},
		},
		"New NetworkPolicy, selector no match": {
			netpol: createNetworkPolicy("1", "default", map[string]string{
				"app": "no-match",
			}),
			oldNetpol:           nil,
			action:              central.ResourceAction_CREATE_RESOURCE,
			expectedEvents:      nil,
			expectedDeployments: []*deploymentWrap{},
		},
		"New NetworkPolicy, namespace with no deployments": {
			netpol: createNetworkPolicy("1", "random-namespace", map[string]string{
				"app": "sensor",
			}),
			oldNetpol:           nil,
			action:              central.ResourceAction_CREATE_RESOURCE,
			expectedEvents:      nil,
			expectedDeployments: []*deploymentWrap{},
		},
		"Update NetworkPolicy": {
			netpol: createNetworkPolicy("1", "default", map[string]string{
				"app":  "sensor",
				"role": "backend",
			}),
			oldNetpol: createNetworkPolicy("1", "default", map[string]string{
				"app":  "sensor",
				"role": "backend",
			}),
			action:         central.ResourceAction_UPDATE_RESOURCE,
			expectedEvents: nil,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
			},
		},
		"Update NetworkPolicy, no selector": {
			netpol:         createNetworkPolicy("1", "default", nil),
			oldNetpol:      createNetworkPolicy("1", "default", nil),
			action:         central.ResourceAction_UPDATE_RESOURCE,
			expectedEvents: nil,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "2",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "4",
						Namespace: "default",
					},
				},
			},
		},
		"Update NetworkPolicy, new selector": {
			netpol: createNetworkPolicy("1", "default", map[string]string{
				"app":  "sensor",
				"role": "backend",
			}),
			oldNetpol:      createNetworkPolicy("1", "default", nil),
			action:         central.ResourceAction_UPDATE_RESOURCE,
			expectedEvents: nil,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "2",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "4",
						Namespace: "default",
					},
				},
			},
		},
		"Update NetworkPolicy, delete selector": {
			netpol: createNetworkPolicy("1", "default", nil),
			oldNetpol: createNetworkPolicy("1", "default", map[string]string{
				"app":  "sensor",
				"role": "backend",
			}),
			action:         central.ResourceAction_UPDATE_RESOURCE,
			expectedEvents: nil,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "2",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "4",
						Namespace: "default",
					},
				},
			},
		},
		"Update NetworkPolicy, change selector": {
			netpol: createNetworkPolicy("1", "default", map[string]string{
				"app": "sensor-2",
			}),
			oldNetpol: createNetworkPolicy("1", "default", map[string]string{
				"app":  "sensor",
				"role": "backend",
			}),
			action:         central.ResourceAction_UPDATE_RESOURCE,
			expectedEvents: nil,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "4",
						Namespace: "default",
					},
				},
			},
		},
		"Delete NetworkPolicy": {
			netpol: createNetworkPolicy("1", "default", map[string]string{
				"app":  "sensor",
				"role": "backend",
			}),
			oldNetpol:      nil,
			action:         central.ResourceAction_REMOVE_RESOURCE,
			expectedEvents: nil,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
			},
		},
		"Delete NetworkPolicy, no selector": {
			netpol:         createNetworkPolicy("1", "default", nil),
			oldNetpol:      nil,
			action:         central.ResourceAction_REMOVE_RESOURCE,
			expectedEvents: nil,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "2",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "4",
						Namespace: "default",
					},
				},
			},
		},
	}
	for name, c := range cases {
		suite.T().Run(name, func(t *testing.T) {
			c.expectedEvents = createSensorEvent(c.netpol.(*networkingV1.NetworkPolicy), c.action)
			upsertMock := suite.netpolStore.EXPECT().Upsert(gomock.Any()).Return()
			deleteMock := suite.netpolStore.EXPECT().Delete(gomock.Any(), gomock.Any()).Return()
			if c.action == central.ResourceAction_REMOVE_RESOURCE {
				deleteMock.Times(1)
				upsertMock.Times(0)
			} else {
				upsertMock.Times(1)
				deleteMock.Times(0)
			}
			events := suite.dispatcher.ProcessEvent(c.netpol, c.oldNetpol, c.action)
			require.NotNil(t, events)
			for _, e := range events.ForwardMessages {
				_, ok := c.expectedEvents[e.Id]
				assert.Truef(t, ok, "Expected SensorEvent with NetworkPolicy Id %s not found", e.Id)
			}
		})
	}
}
