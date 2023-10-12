package resources

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/deduper"
)

type serviceAccountKey struct {
	namespace, name string
}

// ServiceAccountStore keeps a mapping of service accounts to image pull secrets
type ServiceAccountStore struct {
	lock                        sync.RWMutex
	serviceAccountToPullSecrets map[serviceAccountKey][]string
	serviceAccountIDs           set.StringSet
}

// ReconcileDelete is called after Sensor reconnects with Central and receives its state hashes.
// Reconciliacion ensures that Sensor and Central have the same state by checking whether a given resource
// shall be deleted from Central.
func (sas *ServiceAccountStore) ReconcileDelete(resType, resID string, _ uint64) (string, error) {
	if resType != deduper.TypeServiceAccount.String() {
		return "", nil
	}
	// Resource exists on central but not on Sensor, send delete event
	if !sas.serviceAccountIDs.Contains(resID) {
		return resID, nil
	}
	return "", nil
}

func newServiceAccountStore() *ServiceAccountStore {
	return &ServiceAccountStore{
		serviceAccountToPullSecrets: make(map[serviceAccountKey][]string),
	}
}

func key(namespace, name string) serviceAccountKey {
	return serviceAccountKey{
		namespace: namespace,
		name:      name,
	}
}

// Cleanup deletes all entries from store
func (sas *ServiceAccountStore) Cleanup() {
	sas.lock.Lock()
	defer sas.lock.Unlock()

	sas.serviceAccountToPullSecrets = make(map[serviceAccountKey][]string)
	sas.serviceAccountIDs = set.NewStringSet()
}

// GetImagePullSecrets get the image pull secrets for a namespace and secret name pair
func (sas *ServiceAccountStore) GetImagePullSecrets(namespace, name string) []string {
	sas.lock.RLock()
	defer sas.lock.RUnlock()
	return sas.serviceAccountToPullSecrets[key(namespace, name)]
}

// Add inserts a new service account and its image pull secrets to the map
func (sas *ServiceAccountStore) Add(sa *storage.ServiceAccount) {
	sas.lock.Lock()
	defer sas.lock.Unlock()
	sas.serviceAccountToPullSecrets[key(sa.GetNamespace(), sa.GetName())] = sa.GetImagePullSecrets()
	sas.serviceAccountIDs.Add(sa.Id)
}

// Remove removes the service account from the map
func (sas *ServiceAccountStore) Remove(sa *storage.ServiceAccount) {
	sas.lock.Lock()
	defer sas.lock.Unlock()
	delete(sas.serviceAccountToPullSecrets, key(sa.GetNamespace(), sa.GetName()))
	sas.serviceAccountIDs.Remove(sa.Id)
}
