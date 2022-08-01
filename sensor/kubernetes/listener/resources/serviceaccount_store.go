package resources

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

type serviceAccountKey struct {
	namespace, name string
}

type ServiceAccountStore struct {
	lock                        sync.RWMutex
	serviceAccountToPullSecrets map[serviceAccountKey][]string
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

func (sas *ServiceAccountStore) GetImagePullSecrets(namespace, name string) []string {
	sas.lock.RLock()
	defer sas.lock.RUnlock()
	return sas.serviceAccountToPullSecrets[key(namespace, name)]
}

func (sas *ServiceAccountStore) Add(sa *storage.ServiceAccount) {
	sas.lock.Lock()
	defer sas.lock.Unlock()
	sas.serviceAccountToPullSecrets[key(sa.GetNamespace(), sa.GetName())] = sa.GetImagePullSecrets()
}

func (sas *ServiceAccountStore) Remove(sa *storage.ServiceAccount) {
	sas.lock.Lock()
	defer sas.lock.Unlock()
	delete(sas.serviceAccountToPullSecrets, key(sa.GetNamespace(), sa.GetName()))
}
