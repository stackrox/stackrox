package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	networkingV1 "k8s.io/api/networking/v1"
)

// networkPolicyHandler handles network policy resource events.
type networkPolicyHandler struct{}

func newNetworkPolicyHandler() *networkPolicyHandler {
	return &networkPolicyHandler{}
}

// Process processes a network policy resource event, and returns the sensor events to generate.
func (h *networkPolicyHandler) Process(np *networkingV1.NetworkPolicy, action central.ResourceAction) []*central.SensorEvent {
	return []*central.SensorEvent{
		{
			Id:     string(np.UID),
			Action: action,
			Resource: &central.SensorEvent_NetworkPolicy{
				NetworkPolicy: networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: np}.ToRoxNetworkPolicy(),
			},
		},
	}
}
