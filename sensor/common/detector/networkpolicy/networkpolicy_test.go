package networkpolicy

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stretchr/testify/suite"
)

func TestNetworkPolicy(t *testing.T) {
	suite.Run(t, new(NetworkPolicySuite))
}

type NetworkPolicySuite struct {
	suite.Suite

	networkStore  *mocks.MockNetworkPolicyStore
	networkPolicy *Finder
	mockCtrl      *gomock.Controller
	envIsolator   *envisolator.EnvIsolator
}

var _ suite.SetupTestSuite = (*NetworkPolicySuite)(nil)
var _ suite.TearDownTestSuite = (*NetworkPolicySuite)(nil)

func (suite *NetworkPolicySuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.networkStore = mocks.NewMockNetworkPolicyStore(suite.mockCtrl)
	suite.networkPolicy = &Finder{store: suite.networkStore}
	suite.envIsolator = envisolator.NewEnvIsolator(suite.T())

	suite.envIsolator.Setenv(features.NetworkPolicySystemPolicy.EnvVar(), "true")
}

func (suite *NetworkPolicySuite) TearDownTest() {
	suite.mockCtrl.Finish()
	suite.envIsolator.RestoreAll()
}

func deployment(namespace string, labels map[string]string) *storage.Deployment {
	dep := new(storage.Deployment)
	dep.Namespace = namespace
	dep.Labels = labels
	return dep
}

func policy(classificationEnums []storage.NetworkPolicyType) *storage.NetworkPolicy {
	netpol := new(storage.NetworkPolicy)
	netpol.Spec = new(storage.NetworkPolicySpec)
	netpol.Spec.PolicyTypes = classificationEnums
	return netpol
}

func (suite *NetworkPolicySuite) Test_ReturnNilIfFeatureFlagDisabled() {
	suite.envIsolator.Setenv(features.NetworkPolicySystemPolicy.EnvVar(), "false")
	dep := deployment("", map[string]string{})
	aug := suite.networkPolicy.GetNetworkPoliciesApplied(dep)
	suite.Nil(aug, "augmented object should be nil")
}

func (suite *NetworkPolicySuite) Test_GetNetworkPoliciesApplied() {
	if !features.NetworkPolicySystemPolicy.Enabled() {
		suite.T().Skip()
	}

	cases := map[string]struct {
		policiesInStore         map[string]*storage.NetworkPolicy
		expectedAugmentedObject *augmentedobjs.NetworkPoliciesApplied
	}{
		"No policies for deployment": {
			policiesInStore: map[string]*storage.NetworkPolicy{},
			expectedAugmentedObject: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: false,
				HasEgressNetworkPolicy:  false,
			},
		},
		"Ingress Policy": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
				}),
			},
			expectedAugmentedObject: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: true,
				HasEgressNetworkPolicy:  false,
			},
		},
		"Egress Policy": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE,
				}),
			},
			expectedAugmentedObject: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: false,
				HasEgressNetworkPolicy:  true,
			},
		},
		"Ingress and Egress on same policy object": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
					storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE,
				}),
			},
			expectedAugmentedObject: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: true,
				HasEgressNetworkPolicy:  true,
			},
		},
		"Ingress and Egress on different policy objects": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
				}),
				"id2": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE,
				}),
			},
			expectedAugmentedObject: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: true,
				HasEgressNetworkPolicy:  true,
			},
		},
		"Both missing if policy is UNSET": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_UNSET_NETWORK_POLICY_TYPE,
				}),
			},
			expectedAugmentedObject: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: false,
				HasEgressNetworkPolicy:  false,
			},
		},
	}

	for name, testCase := range cases {
		ns := "example"
		labels := map[string]string{"label1": "value1"}
		dep := deployment(ns, labels)

		suite.Run(name, func() {
			suite.networkStore.EXPECT().
				Find(gomock.Eq(ns), gomock.Eq(labels)).
				Return(testCase.policiesInStore)
			aug := suite.networkPolicy.GetNetworkPoliciesApplied(dep)
			// Assume, that all policies from store would match the given deployment
			testCase.expectedAugmentedObject.Policies = testCase.policiesInStore
			suite.Equal(testCase.expectedAugmentedObject, aug)
		})
	}
}
