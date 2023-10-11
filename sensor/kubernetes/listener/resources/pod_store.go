package resources

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/deduper"
)

// PodStore stores pods (by namespace, deploymentID, and id).
type PodStore struct {
	lock sync.RWMutex
	pods map[string]map[string]map[string]*storage.Pod
}

// ReconcileDelete is called after Sensor reconnects with Central and receives its state hashes.
// Reconciliacion ensures that Sensor and Central have the same state by checking whether a given resource
// shall be deleted from Central.
func (ps *PodStore) ReconcileDelete(resType, resID string, _ uint64) (string, error) {
	if resType != deduper.TypePod.String() {
		return "", nil
	}
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	for _, p := range ps.getAllNoLock() {
		if p.GetId() == resID {
			// Resource exists on central and Sensor, nothing to do
			return "", nil
		}
	}
	// Resource exists on central but not on Sensor, send delete event
	return resID, nil
}

// Cleanup deletes all entries from store
func (ps *PodStore) Cleanup() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.pods = make(map[string]map[string]map[string]*storage.Pod)
}

// newPodStore creates and returns a new pod store.
func newPodStore() *PodStore {
	return &PodStore{
		pods: make(map[string]map[string]map[string]*storage.Pod),
	}
}

func (ps *PodStore) addOrUpdatePod(pod *storage.Pod) {
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

func (ps *PodStore) removePod(namespace, deploymentID, podID string) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	delete(ps.pods[namespace][deploymentID], podID)
}

// forEach takes in a function that will perform some actions for each pod in the given deployment.
// The function MUST NOT update the pods.
func (ps *PodStore) forEach(ns, deploymentID string, f func(*storage.Pod)) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	for _, pod := range ps.pods[ns][deploymentID] {
		f(pod)
	}
}

func (ps *PodStore) getContainersForDeployment(ns, deploymentID string) set.StringSet {
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
func (ps *PodStore) OnNamespaceDeleted(ns string) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	delete(ps.pods, ns)
}

// onDeploymentRemove reacts to a deployment deletion, deleting all pods in this namespace and deployment from the store.
func (ps *PodStore) onDeploymentRemove(wrap *deploymentWrap) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	delete(ps.pods[wrap.GetNamespace()], wrap.GetId())
}

// GetAll returns all pods.
func (ps *PodStore) GetAll() []*storage.Pod {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var ret []*storage.Pod
	for _, pod := range ps.getAllNoLock() {
		ret = append(ret, pod.Clone())
	}
	return ret
}

func (ps *PodStore) getAllNoLock() []*storage.Pod {
	var ret []*storage.Pod
	for _, depMap := range ps.pods {
		for _, podMap := range depMap {
			for _, pod := range podMap {
				ret = append(ret, pod)
			}
		}
	}
	return ret
}

// GetByName returns pod for supplied name in namespace.
func (ps *PodStore) GetByName(podName, namespace string) *storage.Pod {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	depMap := ps.pods[namespace]
	if depMap == nil {
		return nil
	}

	var ret *storage.Pod
	for _, podMap := range depMap {
		for _, pod := range podMap {
			if pod == nil {
				continue
			}
			if pod.GetName() == podName {
				return pod
			}
		}
	}
	return ret
}
