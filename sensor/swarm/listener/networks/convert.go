package networks

import (
	"github.com/docker/docker/api/types"
	"github.com/stackrox/rox/generated/storage"
)

type networkWrap types.NetworkResource

func (n networkWrap) asNetworkPolicy() *storage.NetworkPolicy {
	// Swarm doesn't have network policies so this network policy implements the network segmentation
	// it blocks both ingress and egress out of the network
	return &storage.NetworkPolicy{
		Id:        n.ID,
		Name:      n.Name,
		Namespace: n.Name,
		Spec: &storage.NetworkPolicySpec{
			PodSelector: &storage.LabelSelector{},
			Ingress: []*storage.NetworkPolicyIngressRule{
				{
					From: []*storage.NetworkPolicyPeer{
						{
							PodSelector: &storage.LabelSelector{},
						},
					},
				},
			},
			Egress: []*storage.NetworkPolicyEgressRule{
				{
					To: []*storage.NetworkPolicyPeer{
						{
							PodSelector: &storage.LabelSelector{},
						},
						{
							IpBlock: &storage.IPBlock{
								Cidr: "0.0.0.0/32",
							},
						},
					},
				},
			},
			PolicyTypes: []storage.NetworkPolicyType{
				storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
				storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE,
			},
		},
	}
}
