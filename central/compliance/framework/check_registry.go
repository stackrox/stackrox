package framework

import (
	"github.com/stackrox/stackrox/pkg/sync"
)

// CheckRegistry stores compliance checks, and allows retrieving them by ID.
type CheckRegistry interface {
	Register(check Check) error
	Lookup(id string) Check
	GetAll() []Check
	Delete(id string)
}

type checkRegistry struct {
	lock   sync.RWMutex
	checks map[string]Check
}

var (
	registry     *checkRegistry
	registryInit sync.Once
)

// RegistrySingleton returns the global check registry.
func RegistrySingleton() CheckRegistry {
	registryInit.Do(func() {
		registry = newCheckRegistry()
	})
	return registry
}

func newCheckRegistry() *checkRegistry {
	registry := &checkRegistry{
		checks: make(map[string]Check),
	}
	return registry
}

func (r *checkRegistry) Register(check Check) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.checks[check.ID()] = check
	return nil
}

func (r *checkRegistry) Delete(id string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.checks, id)
}

func (r *checkRegistry) Lookup(id string) Check {
	r.lock.RLock()
	defer r.lock.RUnlock()
	c := r.checks[id]
	if c != nil {
		return c
	}
	return nil
}

func (r *checkRegistry) GetAll() []Check {
	r.lock.RLock()
	defer r.lock.RUnlock()
	result := make([]Check, 0, len(r.checks))
	for _, check := range r.checks {
		result = append(result, check)
	}
	return result
}
