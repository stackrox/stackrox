package graph

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestHasEgress(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		policyTypes []storage.NetworkPolicyType
		expected    bool
	}{
		{
			name:        "no values",
			policyTypes: []storage.NetworkPolicyType{},
			expected:    false,
		},
		{
			name:        "ingress only",
			policyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
			expected:    false,
		},
		{
			name:        "egress only",
			policyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
			expected:    true,
		},
		{
			name:        "ingress + egress only",
			policyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
			expected:    true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, hasEgress(c.policyTypes))
		})
	}
}

func TestHasIngress(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		policyTypes []storage.NetworkPolicyType
		expected    bool
	}{
		{
			name:        "no values",
			policyTypes: []storage.NetworkPolicyType{},
			expected:    true,
		},
		{
			name:        "ingress only",
			policyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
			expected:    true,
		},
		{
			name:        "egress only",
			policyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
			expected:    false,
		},
		{
			name:        "ingress + egress only",
			policyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
			expected:    true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, hasIngress(c.policyTypes))
		})
	}
}
