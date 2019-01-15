package resources

import "k8s.io/apimachinery/pkg/labels"

// deploymentStore stores deployments (by namespace and id).
type deploymentStore struct {
	deployments map[string]map[string]*deploymentWrap
}

// newDeploymentStore creates and returns a new deployment store.
func newDeploymentStore() *deploymentStore {
	return &deploymentStore{
		deployments: make(map[string]map[string]*deploymentWrap),
	}
}

func (ds *deploymentStore) addOrUpdateDeployment(wrap *deploymentWrap) {
	nsMap := ds.deployments[wrap.GetNamespace()]
	if nsMap == nil {
		nsMap = make(map[string]*deploymentWrap)
		ds.deployments[wrap.GetNamespace()] = nsMap
	}
	nsMap[wrap.GetId()] = wrap
}

func (ds *deploymentStore) removeDeployment(wrap *deploymentWrap) {
	nsMap := ds.deployments[wrap.GetNamespace()]
	if nsMap == nil {
		return
	}
	delete(nsMap, wrap.GetId())
}

func (ds *deploymentStore) getOwningDeployments(namespace string, podLabels map[string]string) (owning []*deploymentWrap) {
	podLabelSet := labels.Set(podLabels)
	for _, wrap := range ds.deployments[namespace] {
		if wrap.podSelector != nil && wrap.podSelector.Matches(podLabelSet) {
			owning = append(owning, wrap)
		}
	}
	return
}

func (ds *deploymentStore) getMatchingDeployments(namespace string, sel selector) (matching []*deploymentWrap) {
	for _, wrap := range ds.deployments[namespace] {
		if sel.Matches(labels.Set(wrap.PodLabels)) {
			matching = append(matching, wrap)
		}
	}
	return
}

// OnNamespaceDeleted reacts to a namespace deletion, deleting all deployments in this namespace from the store.
func (ds *deploymentStore) OnNamespaceDeleted(ns string) {
	delete(ds.deployments, ns)
}
