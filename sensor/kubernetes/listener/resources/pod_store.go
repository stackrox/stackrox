package resources

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// PodStore stores pods (by namespace, deploymentID, and id).
type podStore struct {
	lock sync.RWMutex
	pods map[string]map[string]map[string]*storage.Pod
}

// newPodStore creates and returns a new pod store.
func newPodStore() *podStore {
	return &podStore{
		pods: make(map[string]map[string]map[string]*storage.Pod),
	}
}

func (ps *podStore) addOrUpdatePod(pod *storage.Pod) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	nsMap := ps.pods[pod.GetNamespace()]
	if nsMap == nil {
		nsMap = make(map[string]map[string]*storage.Pod)
		ps.pods[pod.GetNamespace()] = nsMap
	}
	dMap := nsMap[pod.GetDeploymentId()]
	if dMap == nil {
		dMap = make(map[string]*storage.Pod)
		nsMap[pod.GetDeploymentId()] = dMap
	}
	dMap[pod.GetId()] = pod
}

func (ps *podStore) removePod(namespace, deploymentID, podID string) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	delete(ps.pods[namespace][deploymentID], podID)
}

// forEach takes in a function that will perform some actions for each pod in the given deployment.
// The function MUST NOT update the pods.
func (ps *podStore) forEach(ns, deploymentID string, f func(*storage.Pod)) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	for _, pod := range ps.pods[ns][deploymentID] {
		f(pod)
	}
}

func (ps *podStore) getContainersForDeployment(ns, deploymentID string) set.StringSet {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	containerIDs := set.NewStringSet()
	for _, pod := range ps.pods[ns][deploymentID] {
		for _, inst := range pod.GetLiveInstances() {
			containerIDs.Add(inst.GetInstanceId().GetId())
		}
	}

	return containerIDs
}

// OnNamespaceDeleted reacts to a namespace deletion, deleting all pods in this namespace from the store.
func (ps *podStore) OnNamespaceDeleted(ns string) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	delete(ps.pods, ns)
}

// onDeploymentRemove reacts to a deployment deletion, deleting all pods in this namespace and deployment from the store.
func (ps *podStore) onDeploymentRemove(wrap *deploymentWrap) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	delete(ps.pods[wrap.GetNamespace()], wrap.GetId())
}
