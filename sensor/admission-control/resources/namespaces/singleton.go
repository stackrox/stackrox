package namespaces

import (
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/admission-control/resources/deployments"
	"github.com/stackrox/rox/sensor/admission-control/resources/pods"
)

var (
	once sync.Once

	nsStore *NamespaceStore
)

func initialize() {
	nsStore = NewNamespaceStore(deployments.Singleton(), pods.Singleton())
}

// Singleton provides the interface for getting annotation values with a datastore backed implementation.
func Singleton() *NamespaceStore {
	once.Do(initialize)
	return nsStore
}
