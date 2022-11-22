package resources

import (
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/rbac"
)

// InMemoryStoreProvider holds all stores used in sensor and exposes a public interface for each that can be used outside of the listeners.
type InMemoryStoreProvider struct {
	serviceStore *serviceStore
	rbacStore    rbac.Store
}

// InitializeStore creates the store instances
func InitializeStore() *InMemoryStoreProvider {
	return &InMemoryStoreProvider{
		serviceStore: newServiceStore(),
		rbacStore:    rbac.NewStore(),
	}
}

// Services returns the service store public interface
func (p *InMemoryStoreProvider) Services() store.ServiceStore {
	return p.serviceStore
}

// RBAC returns the RBAC store public interface
func (p *InMemoryStoreProvider) RBAC() store.RBACStore {
	return p.rbacStore
}
