package installmethod

import (
	storage "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	installMethod string
	mutex         sync.RWMutex
)

// Get returns the installation method.
func Get() string {
	mutex.RLock()
	defer mutex.RUnlock()

	return installMethod
}

// Set sets the installation method based on the managedBy property.
func Set(value storage.ManagerType) {
	mutex.Lock()
	defer mutex.Unlock()

	switch value {
	case storage.ManagerType_MANAGER_TYPE_HELM_CHART:
		installMethod = "helm"
	case storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR:
		installMethod = "operator"
	default:
		installMethod = "manifest"
	}
}
