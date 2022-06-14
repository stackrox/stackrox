package generator

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestCheckPolicyType_IngressPolicy(t *testing.T) {
	t.Parallel()

	ingressPolicy := &storage.NetworkPolicy{
		Spec: &storage.NetworkPolicySpec{
			PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
		},
	}

	assert.True(t, checkPolicyType(ingressPolicy, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE))
	assert.False(t, checkPolicyType(ingressPolicy, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE))
}

func TestCheckPolicyType_EgressPolicy(t *testing.T) {
	t.Parallel()

	egressPolicy := &storage.NetworkPolicy{
		Spec: &storage.NetworkPolicySpec{
			PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
		},
	}

	assert.False(t, checkPolicyType(egressPolicy, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE))
	assert.True(t, checkPolicyType(egressPolicy, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE))
}

func TestCheckPolicyType_IngressEgressPolicy(t *testing.T) {
	t.Parallel()

	ingressEgressPolicy := &storage.NetworkPolicy{
		Spec: &storage.NetworkPolicySpec{
			PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
		},
	}

	assert.True(t, checkPolicyType(ingressEgressPolicy, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE))
	assert.True(t, checkPolicyType(ingressEgressPolicy, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE))
}

func TestGroupNetworkPolicies(t *testing.T) {
	t.Parallel()

	policy1 := &storage.NetworkPolicy{
		Id:        "policy1",
		Namespace: "ns1",
		Spec: &storage.NetworkPolicySpec{
			PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
		},
	}
	policy2 := &storage.NetworkPolicy{
		Id:        "policy2",
		Namespace: "ns1",
		Spec: &storage.NetworkPolicySpec{
			PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
		},
	}
	policy3 := &storage.NetworkPolicy{
		Id:        "policy3",
		Namespace: "ns1",
		Spec: &storage.NetworkPolicySpec{
			PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
		},
	}
	policy4 := &storage.NetworkPolicy{
		Id:        "policy4",
		Namespace: "ns2",
		Spec: &storage.NetworkPolicySpec{
			PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
		},
	}

	ingressPolicies, egressPolicies := groupNetworkPolicies([]*storage.NetworkPolicy{policy1, policy2, policy3, policy4})

	assert.Len(t, ingressPolicies, 1)
	assert.Len(t, egressPolicies, 2)
	assert.ElementsMatch(t, []*storage.NetworkPolicy{policy1, policy3}, ingressPolicies["ns1"])
	assert.ElementsMatch(t, []*storage.NetworkPolicy{policy2, policy3}, egressPolicies["ns1"])
	assert.Empty(t, ingressPolicies["ns2"])
	assert.ElementsMatch(t, []*storage.NetworkPolicy{policy4}, egressPolicies["ns2"])
}

func TestHasMatchingPolicy_Success(t *testing.T) {
	t.Parallel()

	policies := []*storage.NetworkPolicy{
		{
			Namespace: "ns1",
			Spec: &storage.NetworkPolicySpec{
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"app": "foo"},
				},
			},
		},
	}

	deployment := &storage.Deployment{
		Namespace: "ns1",
		PodLabels: map[string]string{"app": "foo"},
	}

	assert.True(t, hasMatchingPolicy(deployment, policies))
}

func TestHasMatchingPolicy_WrongNamespace(t *testing.T) {
	t.Parallel()

	policies := []*storage.NetworkPolicy{
		{
			Namespace: "ns2",
			Spec: &storage.NetworkPolicySpec{
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"app": "foo"},
				},
			},
		},
	}

	deployment := &storage.Deployment{
		Namespace: "ns1",
		PodLabels: map[string]string{"app": "foo"},
	}

	assert.False(t, hasMatchingPolicy(deployment, policies))
}

func TestHasMatchingPolicy_WrongTypeOfLabels(t *testing.T) {
	t.Parallel()

	policies := []*storage.NetworkPolicy{
		{
			Namespace: "ns1",
			Spec: &storage.NetworkPolicySpec{
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"app": "foo"},
				},
			},
		},
	}

	deployment := &storage.Deployment{
		Namespace: "ns1",
		Labels:    map[string]string{"app": "foo"},
	}

	assert.False(t, hasMatchingPolicy(deployment, policies))
}
