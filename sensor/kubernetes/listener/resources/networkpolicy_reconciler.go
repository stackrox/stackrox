package resources

type networkPolicyReconciler interface {
	UpdateNetworkPolicyForDeployment(deployment *deploymentWrap)
}

type networkPolicyReconcilerImpl struct {
	deploymentStore *DeploymentStore
	netpolStore     networkPolicyStore
}

func newNetworkPolicyReconciler(deploymentStore *DeploymentStore, netpolStore networkPolicyStore) networkPolicyReconciler {
	return &networkPolicyReconcilerImpl{
		deploymentStore: deploymentStore,
		netpolStore:     netpolStore,
	}
}

func (n *networkPolicyReconcilerImpl) UpdateNetworkPolicyForDeployment(deployment *deploymentWrap) {
	cloned := deployment.Clone()
	netpols := n.netpolStore.Find(cloned.GetNamespace(), cloned.PodLabels)
	for _, np := range netpols {
		// TODO: update cloned.networkPolicyInformation
		cloned.networkPoliciesApplied.MissingIngressNetworkPolicy = np.Spec.GetIngress() == nil && cloned.networkPoliciesApplied.MissingIngressNetworkPolicy
	}
	n.deploymentStore.addOrUpdateDeployment(cloned)
}
