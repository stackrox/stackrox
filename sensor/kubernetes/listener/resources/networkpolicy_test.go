package resources

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/labels"
)

func TestNetworkPolicyDispatcher(t *testing.T) {
	suite.Run(t, new(NetworkPolicyDispatcherSuite))
}

type NetworkPolicyDispatcherSuite struct {
	suite.Suite

	mockCtrl        *gomock.Controller
	netpolStore     networkPolicyStore
	deploymentStore *DeploymentStore
	detector        *mocks.MockDetector
	dispatcher      *networkPolicyDispatcher

	envIsolator *envisolator.EnvIsolator
}

var _ suite.SetupTestSuite = (*NetworkPolicyDispatcherSuite)(nil)
var _ suite.TearDownTestSuite = (*NetworkPolicyDispatcherSuite)(nil)

func (suite *NetworkPolicyDispatcherSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.netpolStore = newNetworkPoliciesStore()
	suite.deploymentStore = newDeploymentStore()
	suite.detector = mocks.NewMockDetector(suite.mockCtrl)

	suite.dispatcher = newNetworkPolicyDispatcher(suite.netpolStore, suite.deploymentStore, suite.detector)

	suite.envIsolator = envisolator.NewEnvIsolator(suite.T())

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

	suite.envIsolator.Setenv(features.NetworkPolicySystemPolicy.EnvVar(), "true")
}

func (suite *NetworkPolicyDispatcherSuite) TearDownTest() {
	suite.mockCtrl.Finish()
	suite.envIsolator.RestoreAll()
}

func createNetworkPolicy(id, namespace string, podSelector map[string]string) *storage.NetworkPolicy {
	netpol := &storage.NetworkPolicy{
		Id:        id,
		Namespace: namespace,
	}
	if netpol == nil || len(podSelector) > 0 {
		netpol.Spec = &storage.NetworkPolicySpec{
			PodSelector: &storage.LabelSelector{
				MatchLabels: podSelector,
			},
		}
	}
	return netpol
}

func (suite *NetworkPolicyDispatcherSuite) Test_GetSelector() {
	if !features.NetworkPolicySystemPolicy.Enabled() {
		suite.T().Skipf("Skipping test since the %s variable is not set", features.NetworkPolicySystemPolicy.EnvVar())
	}

	cases := map[string]struct {
		netpol           *storage.NetworkPolicy
		oldNetpol        *storage.NetworkPolicy
		action           central.ResourceAction
		expectedSelector []map[string]string
		expectedEmpty    bool
	}{
		"New NetworkPolicy": {
			netpol: createNetworkPolicy("1", "default",
				map[string]string{
					"app":  "sensor",
					"role": "backend"}),
			oldNetpol: nil,
			action:    central.ResourceAction_CREATE_RESOURCE,
			expectedSelector: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
			},
			expectedEmpty: false,
		},
		"New NetworkPolicy, no selector": {
			netpol:           createNetworkPolicy("1", "default", nil),
			oldNetpol:        nil,
			action:           central.ResourceAction_CREATE_RESOURCE,
			expectedSelector: []map[string]string{},
			expectedEmpty:    true,
		},
		"Update NetworkPolicy": {
			netpol: createNetworkPolicy("1", "default",
				map[string]string{
					"app":  "sensor",
					"role": "backend"}),
			oldNetpol: createNetworkPolicy("1", "default",
				map[string]string{
					"app":  "sensor-2",
					"role": "backend"}),
			action: central.ResourceAction_UPDATE_RESOURCE,
			expectedSelector: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
				{
					"app":  "sensor-2",
					"role": "backend",
				},
			},
			expectedEmpty: false,
		},
		"Update NetworkPolicy, no selector": {
			netpol:           createNetworkPolicy("1", "default", nil),
			oldNetpol:        createNetworkPolicy("1", "default", nil),
			action:           central.ResourceAction_UPDATE_RESOURCE,
			expectedSelector: []map[string]string{},
			expectedEmpty:    true,
		},
		"Update NetworkPolicy, new selector": {
			netpol: createNetworkPolicy("1", "default",
				map[string]string{
					"app":  "sensor",
					"role": "backend"}),
			oldNetpol: createNetworkPolicy("1", "default", nil),
			action:    central.ResourceAction_UPDATE_RESOURCE,
			expectedSelector: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
			},
			expectedEmpty: true,
		},
		"Update NetworkPolicy, delete selector": {
			netpol: createNetworkPolicy("1", "default", nil),
			oldNetpol: createNetworkPolicy("1", "default",
				map[string]string{
					"app":  "sensor",
					"role": "backend"}),
			action: central.ResourceAction_UPDATE_RESOURCE,
			expectedSelector: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
			},
			expectedEmpty: true,
		},
		"Delete NetworkPolicy": {
			netpol: createNetworkPolicy("1", "default",
				map[string]string{
					"app":  "sensor",
					"role": "backend"}),
			oldNetpol: createNetworkPolicy("1", "default",
				map[string]string{
					"app":  "sensor",
					"role": "backend"}),
			action: central.ResourceAction_REMOVE_RESOURCE,
			expectedSelector: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
			},
			expectedEmpty: false,
		},
		"Delete NetworkPolicy, no selector": {
			netpol:           createNetworkPolicy("1", "default", nil),
			oldNetpol:        createNetworkPolicy("1", "default", nil),
			action:           central.ResourceAction_REMOVE_RESOURCE,
			expectedSelector: []map[string]string{},
			expectedEmpty:    true,
		},
	}
	for name, c := range cases {
		suite.T().Run(name, func(t *testing.T) {
			sel, isEmpty := suite.dispatcher.getSelector(c.netpol, c.oldNetpol, c.action)
			assert.Equal(t, isEmpty, c.expectedEmpty)
			for _, s := range c.expectedSelector {
				assert.True(t, sel.Matches(labels.Set(s)))
			}
		})
	}
}

func (suite *NetworkPolicyDispatcherSuite) Test_UpdateDeploymentsFromStore() {
	if !features.NetworkPolicySystemPolicy.Enabled() {
		suite.T().Skipf("Skipping test since the %s variable is not set", features.NetworkPolicySystemPolicy.EnvVar())
	}

	cases := map[string]struct {
		netpol              *storage.NetworkPolicy
		sel                 []map[string]string
		isEmpty             bool
		expectedDeployments []*deploymentWrap
	}{
		"New NetworkPolicy": {
			netpol: createNetworkPolicy("1", "default", nil),
			sel: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
			},
			isEmpty: false,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
			},
		},
		"Empty selector": {
			netpol:  createNetworkPolicy("1", "default", nil),
			sel:     []map[string]string{},
			isEmpty: true,
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
		"Selector with no deployments": {
			netpol: createNetworkPolicy("1", "default", nil),
			sel: []map[string]string{
				{
					"app": "central",
				},
			},
			isEmpty:             false,
			expectedDeployments: []*deploymentWrap{},
		},
		"Namespace with no deployments": {
			netpol: createNetworkPolicy("1", "random-namespace", nil),
			sel: []map[string]string{
				{
					"app": "sensor",
				},
			},
			isEmpty:             false,
			expectedDeployments: []*deploymentWrap{},
		},
		"Namespace with no deployments, no selector": {
			netpol:              createNetworkPolicy("1", "random-namespace", nil),
			sel:                 []map[string]string{},
			isEmpty:             true,
			expectedDeployments: []*deploymentWrap{},
		},
		"Disjunction selector": {
			netpol: createNetworkPolicy("1", "default", nil),
			sel: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
				{
					"app": "sensor-2",
				},
			},
			isEmpty: false,
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
		"Disjunction selector, with empty member": {
			netpol: createNetworkPolicy("1", "default", nil),
			sel: []map[string]string{
				{},
				{
					"app": "sensor-2",
				},
			},
			isEmpty: true, // If one of the members of the selector is empty the selector is considered empty
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

			deps := map[string]*deploymentWrap{}
			processDeploymentMock := suite.detector.EXPECT().ProcessDeployment(gomock.Any(), gomock.Eq(central.ResourceAction_UPDATE_RESOURCE)).DoAndReturn(func(d *storage.Deployment, _ central.ResourceAction) {
				deps[d.GetId()] = &deploymentWrap{
					Deployment: d,
				}
			})
			processDeploymentMock.Times(len(c.expectedDeployments))
			var sel selector
			for _, s := range c.sel {
				if sel != nil {
					sel = or(sel, SelectorFromMap(s))
				} else {
					sel = SelectorFromMap(s)
				}
			}
			suite.dispatcher.updateDeploymentsFromStore(c.netpol, sel, c.isEmpty)
			for _, d := range c.expectedDeployments {
				_, ok := deps[d.GetId()]
				assert.True(t, ok)
			}
		})
	}
}
