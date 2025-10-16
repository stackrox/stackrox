package generator

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestCheckPolicyType_IngressPolicy(t *testing.T) {

	nps := &storage.NetworkPolicySpec{}
	nps.SetPolicyTypes([]storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE})
	ingressPolicy := &storage.NetworkPolicy{}
	ingressPolicy.SetSpec(nps)

	assert.True(t, checkPolicyType(ingressPolicy, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE))
	assert.False(t, checkPolicyType(ingressPolicy, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE))
}

func TestCheckPolicyType_EgressPolicy(t *testing.T) {

	nps := &storage.NetworkPolicySpec{}
	nps.SetPolicyTypes([]storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE})
	egressPolicy := &storage.NetworkPolicy{}
	egressPolicy.SetSpec(nps)

	assert.False(t, checkPolicyType(egressPolicy, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE))
	assert.True(t, checkPolicyType(egressPolicy, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE))
}

func TestCheckPolicyType_IngressEgressPolicy(t *testing.T) {

	nps := &storage.NetworkPolicySpec{}
	nps.SetPolicyTypes([]storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE})
	ingressEgressPolicy := &storage.NetworkPolicy{}
	ingressEgressPolicy.SetSpec(nps)

	assert.True(t, checkPolicyType(ingressEgressPolicy, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE))
	assert.True(t, checkPolicyType(ingressEgressPolicy, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE))
}

func TestGroupNetworkPolicies(t *testing.T) {

	nps := &storage.NetworkPolicySpec{}
	nps.SetPolicyTypes([]storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE})
	policy1 := &storage.NetworkPolicy{}
	policy1.SetId("policy1")
	policy1.SetNamespace("ns1")
	policy1.SetSpec(nps)
	nps2 := &storage.NetworkPolicySpec{}
	nps2.SetPolicyTypes([]storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE})
	policy2 := &storage.NetworkPolicy{}
	policy2.SetId("policy2")
	policy2.SetNamespace("ns1")
	policy2.SetSpec(nps2)
	nps3 := &storage.NetworkPolicySpec{}
	nps3.SetPolicyTypes([]storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE})
	policy3 := &storage.NetworkPolicy{}
	policy3.SetId("policy3")
	policy3.SetNamespace("ns1")
	policy3.SetSpec(nps3)
	nps4 := &storage.NetworkPolicySpec{}
	nps4.SetPolicyTypes([]storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE})
	policy4 := &storage.NetworkPolicy{}
	policy4.SetId("policy4")
	policy4.SetNamespace("ns2")
	policy4.SetSpec(nps4)

	ingressPolicies, egressPolicies := groupNetworkPolicies([]*storage.NetworkPolicy{policy1, policy2, policy3, policy4})

	assert.Len(t, ingressPolicies, 1)
	assert.Len(t, egressPolicies, 2)
	protoassert.ElementsMatch(t, []*storage.NetworkPolicy{policy1, policy3}, ingressPolicies["ns1"])
	protoassert.ElementsMatch(t, []*storage.NetworkPolicy{policy2, policy3}, egressPolicies["ns1"])
	assert.Empty(t, ingressPolicies["ns2"])
	protoassert.ElementsMatch(t, []*storage.NetworkPolicy{policy4}, egressPolicies["ns2"])
}

func TestHasMatchingPolicy_Success(t *testing.T) {

	policies := []*storage.NetworkPolicy{
		storage.NetworkPolicy_builder{
			Namespace: "ns1",
			Spec: storage.NetworkPolicySpec_builder{
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"app": "foo"},
				}.Build(),
			}.Build(),
		}.Build(),
	}

	deployment := &storage.Deployment{}
	deployment.SetNamespace("ns1")
	deployment.SetPodLabels(map[string]string{"app": "foo"})

	assert.True(t, hasMatchingPolicy(deployment, policies))
}

func TestHasMatchingPolicy_WrongNamespace(t *testing.T) {

	policies := []*storage.NetworkPolicy{
		storage.NetworkPolicy_builder{
			Namespace: "ns2",
			Spec: storage.NetworkPolicySpec_builder{
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"app": "foo"},
				}.Build(),
			}.Build(),
		}.Build(),
	}

	deployment := &storage.Deployment{}
	deployment.SetNamespace("ns1")
	deployment.SetPodLabels(map[string]string{"app": "foo"})

	assert.False(t, hasMatchingPolicy(deployment, policies))
}

func TestHasMatchingPolicy_WrongTypeOfLabels(t *testing.T) {

	policies := []*storage.NetworkPolicy{
		storage.NetworkPolicy_builder{
			Namespace: "ns1",
			Spec: storage.NetworkPolicySpec_builder{
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"app": "foo"},
				}.Build(),
			}.Build(),
		}.Build(),
	}

	deployment := &storage.Deployment{}
	deployment.SetNamespace("ns1")
	deployment.SetLabels(map[string]string{"app": "foo"})

	assert.False(t, hasMatchingPolicy(deployment, policies))
}
