package networks

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/docker/docker/api/types"
)

type networkWrap types.NetworkResource

func (n networkWrap) asNetworkPolicy() *v1.NetworkPolicy {
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
					},
				},
			},
		},
	}
}
