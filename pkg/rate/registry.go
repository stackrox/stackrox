package rate

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	registry     = make(map[string]*Limiter)
	registryLock sync.RWMutex
)

// RegisterLimiter creates and registers a rate limiter for the given workload name.
// If a limiter already exists for this workload, the existing one is returned.
// This is safe for concurrent access.
func RegisterLimiter(workloadName string, globalRate float64, bucketCapacity int) (*Limiter, error) {
	registryLock.Lock()
	defer registryLock.Unlock()

	if existing, ok := registry[workloadName]; ok && existing != nil {
		log.Infof("Using existing rate limiter for %s", workloadName)
		return existing, nil
	}

	limiter, err := NewLimiter(workloadName, globalRate, bucketCapacity)
	if err != nil {
		return nil, err
	}
	registry[workloadName] = limiter
	log.Infof("Registered new rate limiter for %s", workloadName)
	return limiter, nil
}

// GetLimiter returns the rate limiter for the given workload name, or nil if not registered.
func GetLimiter(workloadName string) *Limiter {
	registryLock.RLock()
	defer registryLock.RUnlock()
	return registry[workloadName]
}

// OnClientDisconnectAll notifies all registered limiters that a client has disconnected.
// This should be called when a client connection is terminated.
func OnClientDisconnectAll(clientID string) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	for _, limiter := range registry {
		limiter.OnClientDisconnect(clientID)
	}
}

// ResetForTesting clears the registry. Only use in tests. Required due to use of singleton.
func ResetForTesting() {
	registryLock.Lock()
	defer registryLock.Unlock()
	registry = make(map[string]*Limiter)
}
