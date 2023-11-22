package resources

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
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

func newServiceAccountStore() *ServiceAccountStore {
	sas := &ServiceAccountStore{}
	sas.initMaps()
	return sas
}

func key(namespace, name string) serviceAccountKey {
	return serviceAccountKey{
		namespace: namespace,
		name:      name,
	}
}

func (sas *ServiceAccountStore) initMaps() {
	sas.serviceAccountToPullSecrets = make(map[serviceAccountKey][]string)
	sas.serviceAccountIDs = set.NewStringSet()
}

// Cleanup deletes all entries from store
func (sas *ServiceAccountStore) Cleanup() {
	sas.lock.Lock()
	defer sas.lock.Unlock()

	sas.initMaps()
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
