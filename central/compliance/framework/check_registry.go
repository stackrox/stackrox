package framework

import (
	"fmt"

	"github.com/stackrox/rox/pkg/sync"
)

// CheckRegistry stores compliance checks, and allows retrieving them by ID.
type CheckRegistry interface {
	Register(check Check) error
	Lookup(id string) Check
	GetAll() []Check
}

type checkRegistry struct {
	checks      map[string]Check
	checksMutex sync.RWMutex
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
	r.checksMutex.Lock()
	defer r.checksMutex.Unlock()

	if _, ok := r.checks[check.ID()]; ok {
		return fmt.Errorf("check with id %s already registered", check.ID())
	}
	r.checks[check.ID()] = check
	return nil
}

func (r *checkRegistry) Lookup(id string) Check {
	r.checksMutex.RLock()
	defer r.checksMutex.RUnlock()

	c := r.checks[id]
	if c != nil {
		return c
	}
	return nil
}

func (r *checkRegistry) GetAll() []Check {
	r.checksMutex.RLock()
	defer r.checksMutex.RUnlock()

	result := make([]Check, 0, len(r.checks))
	for _, check := range r.checks {
		result = append(result, check)
	}
	return result
}
