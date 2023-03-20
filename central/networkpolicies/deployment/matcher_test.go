package deployment

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	networkPolicyMocks "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	IngressType          = []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE}
	EgressType           = []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE}
	IngressAndEgressType = []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE}
)

func givenNetworkPolicy(id, cluster, namespace string, types []storage.NetworkPolicyType, podSelector *storage.LabelSelector) *storage.NetworkPolicy {
	return &storage.NetworkPolicy{
		Id:        id,
		ClusterId: cluster,
		Namespace: namespace,
		Spec: &storage.NetworkPolicySpec{
			PodSelector: podSelector,
			PolicyTypes: types,
		},
	}
}

func givenPodSelector(key, value string) *storage.LabelSelector {
	return &storage.LabelSelector{
		MatchLabels: map[string]string{
			key: value,
		},
	}
}

func givenDeployment(id, cluster, namespace string, labels map[string]string) *storage.Deployment {
	return &storage.Deployment{
		Id:        id,
		ClusterId: cluster,
		Namespace: namespace,
		PodLabels: labels,
	}
}

func Test_MatchDeployments(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockNetpol := networkPolicyMocks.NewMockDataStore(mockCtrl)

	mockNetpol.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Eq(fixtureconsts.Cluster1), gomock.Eq("ns1")).
		Return([]*storage.NetworkPolicy{
			givenNetworkPolicy(fixtureconsts.NetworkPolicy1, fixtureconsts.Cluster1, "ns1", IngressType,
				givenPodSelector("selector", "A")),
			givenNetworkPolicy(fixtureconsts.NetworkPolicy2, fixtureconsts.Cluster1, "ns1", EgressType,
				givenPodSelector("selector", "B")),
			givenNetworkPolicy(fixtureconsts.NetworkPolicy3, fixtureconsts.Cluster1, "ns1", IngressAndEgressType,
				givenPodSelector("selector", "C")),
		}, nil)

	mockNetpol.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Eq(fixtureconsts.Cluster1), gomock.Eq("ns2")).
		Return([]*storage.NetworkPolicy{
			givenNetworkPolicy(fixtureconsts.NetworkPolicy1, fixtureconsts.Cluster1, "ns2", IngressType,
				givenPodSelector("selectorA", "A")),
			givenNetworkPolicy(fixtureconsts.NetworkPolicy2, fixtureconsts.Cluster1, "ns2", EgressType,
				givenPodSelector("selectorB", "B")),
		}, nil)

	mockNetpol.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Eq(fixtureconsts.Cluster2), gomock.Eq("ns3")).
		Return([]*storage.NetworkPolicy{
			givenNetworkPolicy(fixtureconsts.NetworkPolicy1, fixtureconsts.Cluster2, "ns3", IngressType,
				givenPodSelector("never", "match")),
		}, nil)

	ctx := context.Background()
	matcher, err := BuildMatcher(ctx, mockNetpol, []ClusterNamespace{
		{
			Cluster:   fixtureconsts.Cluster1,
			Namespace: "ns1",
		},
		{
			Cluster:   fixtureconsts.Cluster1,
			Namespace: "ns2",
		},
		{
			Cluster:   fixtureconsts.Cluster2,
			Namespace: "ns3",
		},
	})

	require.NoError(t, err)

	testCases := map[string]struct {
		deployment                      LabeledResource
		ingressIsolated, egressIsolated bool
		hasPolicyIds                    []string
	}{
		"Match one ingress policy": {
			deployment: givenDeployment(fixtureconsts.Deployment1, fixtureconsts.Cluster1, "ns1", map[string]string{
				"selector": "A",
			}),
			ingressIsolated: true,
			egressIsolated:  false,
			hasPolicyIds:    []string{fixtureconsts.NetworkPolicy1},
		},
		"Match one egress policy": {
			deployment: givenDeployment(fixtureconsts.Deployment2, fixtureconsts.Cluster1, "ns1", map[string]string{
				"selector": "B",
			}),
			ingressIsolated: false,
			egressIsolated:  true,
			hasPolicyIds:    []string{fixtureconsts.NetworkPolicy2},
		},
		"Match one ingress/egress policy": {
			deployment: givenDeployment(fixtureconsts.Deployment3, fixtureconsts.Cluster1, "ns1", map[string]string{
				"selector": "C",
			}),
			ingressIsolated: true,
			egressIsolated:  true,
			hasPolicyIds:    []string{fixtureconsts.NetworkPolicy3},
		},
		"Match two policies": {
			deployment: givenDeployment(fixtureconsts.Deployment4, fixtureconsts.Cluster1, "ns2", map[string]string{
				"selectorA": "A",
				"selectorB": "B",
			}),
			ingressIsolated: true,
			egressIsolated:  true,
			hasPolicyIds:    []string{fixtureconsts.NetworkPolicy1, fixtureconsts.NetworkPolicy2},
		},
		"No policies matched": {
			deployment: givenDeployment(fixtureconsts.Deployment5, fixtureconsts.Cluster2, "ns3", map[string]string{
				"app": "no isolation",
			}),
			ingressIsolated: false,
			egressIsolated:  false,
			hasPolicyIds:    []string{},
		},
		"No policies matched (namespace out of scope)": {
			deployment: givenDeployment(fixtureconsts.Deployment6, fixtureconsts.Cluster1, "random", map[string]string{
				"app": "no isolation",
			}),
			ingressIsolated: false,
			egressIsolated:  false,
			hasPolicyIds:    []string{},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			details := matcher.GetIsolationDetails(testCase.deployment)
			assert.Equal(t, testCase.ingressIsolated, details.IngressIsolated)
			assert.Equal(t, testCase.egressIsolated, details.EgressIsolated)
			for _, policyID := range testCase.hasPolicyIds {
				assert.Contains(t, details.PolicyIDs, policyID)
			}
		})
	}
}
