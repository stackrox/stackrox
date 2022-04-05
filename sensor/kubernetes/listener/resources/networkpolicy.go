package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/store"
	networkingV1 "k8s.io/api/networking/v1"
)

// networkPolicyDispatcher handles network policy resource events.
type networkPolicyDispatcher struct {
	netpolStore     store.NetworkPolicyStore
	deploymentStore *DeploymentStore
	detector        detector.Detector
}

func newNetworkPolicyDispatcher(networkPolicyStore store.NetworkPolicyStore, deploymentStore *DeploymentStore, detector detector.Detector) *networkPolicyDispatcher {
	return &networkPolicyDispatcher{
		netpolStore:     networkPolicyStore,
		deploymentStore: deploymentStore,
		detector:        detector,
	}
}

// Process processes a network policy resource event, and returns the sensor events to generate.
func (h *networkPolicyDispatcher) ProcessEvent(obj, old interface{}, action central.ResourceAction) []*central.SensorEvent {
	np := obj.(*networkingV1.NetworkPolicy)

	roxNetpol := networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: np}.ToRoxNetworkPolicy()

	if features.NetworkPolicySystemPolicy.Enabled() {
		var roxOldNetpol *storage.NetworkPolicy
		if old != nil {
			oldNp := old.(*networkingV1.NetworkPolicy)
			roxOldNetpol = networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: oldNp}.ToRoxNetworkPolicy()
		}
		sel, matchesAll := h.getSelector(roxNetpol, roxOldNetpol, action)
		if action == central.ResourceAction_REMOVE_RESOURCE {
			h.netpolStore.Delete(roxNetpol.GetId(), roxNetpol.GetNamespace())
		} else {
			h.netpolStore.Upsert(roxNetpol)
		}

		h.updateDeploymentsFromStore(roxNetpol, sel, matchesAll)
	}

	return []*central.SensorEvent{
		{
			Id:     string(np.UID),
			Action: action,
			Resource: &central.SensorEvent_NetworkPolicy{
				NetworkPolicy: roxNetpol,
			},
		},
	}
}

func (h *networkPolicyDispatcher) getSelector(np, oldNp *storage.NetworkPolicy, action central.ResourceAction) (selector, bool) {
	var sel selector
	// If the selector is empty the Network Policies match all the deployments in the namespace
	// We cannot use the Selector's method Empty() because nothingSelector returns false in this case which is the opposite of what we want
	matchesAll := true

	if oldNp != nil {
		matchLabels := oldNp.GetSpec().GetPodSelector().GetMatchLabels()
		sel = SelectorFromMap(matchLabels)
		matchesAll = len(matchLabels) == 0
	}

	matchLabels := np.GetSpec().GetPodSelector().GetMatchLabels()
	if action == central.ResourceAction_UPDATE_RESOURCE {
		if sel != nil {
			sel = or(sel, SelectorFromMap(matchLabels))
			matchesAll = matchesAll || len(matchLabels) == 0
		} else {
			sel = SelectorFromMap(matchLabels)
			matchesAll = len(matchLabels) == 0
		}
	} else if action == central.ResourceAction_CREATE_RESOURCE {
		sel = SelectorFromMap(matchLabels)
		matchesAll = len(matchLabels) == 0
	}
	return sel, matchesAll
}

func (h *networkPolicyDispatcher) updateDeploymentsFromStore(np *storage.NetworkPolicy, sel selector, matchesAll bool) {
	var deployments []*deploymentWrap
	if matchesAll {
		// Network Policies with no selector match with all the deployments in the namespace
		deployments = h.deploymentStore.getAllDeploymentsInNamespace(np.GetNamespace())
	} else {
		deployments = h.deploymentStore.getMatchingDeployments(np.GetNamespace(), sel)
	}
	for _, deploymentWrap := range deployments {
		h.detector.ProcessDeployment(deploymentWrap.GetDeployment(), central.ResourceAction_UPDATE_RESOURCE)
	}
}
