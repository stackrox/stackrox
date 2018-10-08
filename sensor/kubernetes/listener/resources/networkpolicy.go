package resources

import (
	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	networkingV1 "k8s.io/api/networking/v1"
)

// networkPolicyHandler handles network policy resource events.
type networkPolicyHandler struct{}

func newNetworkPolicyHandler() *networkPolicyHandler {
	return &networkPolicyHandler{}
}

// Process processes a network policy resource event, and returns the sensor events to generate.
func (h *networkPolicyHandler) Process(np *networkingV1.NetworkPolicy, action pkgV1.ResourceAction) []*pkgV1.SensorEvent {
	return []*pkgV1.SensorEvent{
		{
			Id:     string(np.UID),
			Action: action,
			Resource: &pkgV1.SensorEvent_NetworkPolicy{
				NetworkPolicy: networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: np}.ToRoxNetworkPolicy(),
			},
		},
	}
}
