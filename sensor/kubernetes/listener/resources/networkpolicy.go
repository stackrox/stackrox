package resources

import (
  "fmt"

  "github.com/stackrox/rox/generated/internalapi/central"
  networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
  networkingV1 "k8s.io/api/networking/v1"
)

// networkPolicyDispatcher handles network policy resource events.
type networkPolicyDispatcher struct {
	npStore *NetworkPolicyStore
}

func newNetworkPolicyDispatcher(store *NetworkPolicyStore) *networkPolicyDispatcher {
	return &networkPolicyDispatcher{
		npStore: store,
	}
}

// ProcessEvent processes a network policy resource event, and returns the sensor events to generate.
func (h *networkPolicyDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) []*central.SensorEvent {
  k8sNP := obj.(*networkingV1.NetworkPolicy)
  netPolicy := networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: k8sNP}.ToRoxNetworkPolicy()

  switch action {
  case central.ResourceAction_CREATE_RESOURCE:
    fmt.Printf("Adding NetworkPolicy '%s' to store\n", k8sNP.Name)
    h.npStore.addNetPolicy(netPolicy)
  case central.ResourceAction_REMOVE_RESOURCE:
    fmt.Printf("Deleting NetworkPolicy '%s' from store\n", k8sNP.Name)
    h.npStore.deleteNetPolicy(netPolicy)
  case central.ResourceAction_UPDATE_RESOURCE:
    fmt.Printf("Updating NetworkPolicy '%s' in store\n", k8sNP.Name)
    h.npStore.update(netPolicy)
  default:
    fmt.Printf("Unknown action to NetworkPolicy '%s': %v\n", k8sNP.Name, action)
  }

  return []*central.SensorEvent{
    {
      Id:     string(k8sNP.UID),
      Action: action,
      Resource: &central.SensorEvent_NetworkPolicy{
        NetworkPolicy: networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: k8sNP}.ToRoxNetworkPolicy(),
      },
    },
  }
}

