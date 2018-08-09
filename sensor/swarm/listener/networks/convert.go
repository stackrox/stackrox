package networks

import (
	"github.com/docker/docker/api/types"
	"github.com/stackrox/rox/generated/api/v1"
)

type networkWrap types.NetworkResource

func (n networkWrap) asNetworkPolicy() *v1.NetworkPolicy {
	// Swarm doesn't have network policies so this network policy implements the network segmentation
	// it blocks both ingress and egress out of the network
	return &v1.NetworkPolicy{
		Id:        n.ID,
		Name:      n.Name,
		Namespace: n.Name,
		Spec: &v1.NetworkPolicySpec{
			PodSelector: &v1.LabelSelector{},
			Ingress: []*v1.NetworkPolicyIngressRule{
				{
					From: []*v1.NetworkPolicyPeer{
						{
							PodSelector: &v1.LabelSelector{},
						},
					},
				},
			},
			Egress: []*v1.NetworkPolicyEgressRule{
				{
					To: []*v1.NetworkPolicyPeer{
						{
							PodSelector: &v1.LabelSelector{},
						},
						{
							IpBlock: &v1.IPBlock{
								Cidr: "0.0.0.0/32",
							},
						},
					},
				},
			},
			PolicyTypes: []v1.NetworkPolicyType{
				v1.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
				v1.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE,
			},
		},
	}
}
