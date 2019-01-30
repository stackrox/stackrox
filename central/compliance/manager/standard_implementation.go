package manager

import (
	"fmt"
	"sync"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
)

// StandardImplementation is the implementation of a compliance standard, i.e., the list of checks implementing all or
// a subset of the controls of the standard.
type StandardImplementation struct {
	Standard *standards.Standard
	Checks   []framework.Check
}

// StandardImplementationStore contains the standard implementations for all compliance standards. The interface exists
// for mocking/testing purposes only.
type StandardImplementationStore interface {
	ListStandardImplementations() []StandardImplementation
	LookupStandardImplementation(standardID string) (StandardImplementation, error)
}

var (
	defaultStandardImplStoreInstance     StandardImplementationStore
	defaultStandardImplStoreInstanceInit sync.Once
)

type standardImplStore struct {
	standardImplsByID map[string]StandardImplementation
}

func createDefaultStandardImplStore() *standardImplStore {
	store := &standardImplStore{
		standardImplsByID: make(map[string]StandardImplementation),
	}

	for _, standard := range standards.RegistrySingleton().AllStandards() {
		checks := getChecksForStandard(standard)
		store.standardImplsByID[standard.ID] = StandardImplementation{
			Standard: standard,
			Checks:   checks,
		}
		log.Infof("Compliance standard %s: found checks for %d/%d controls", standard.Name, len(checks), len(standard.AllControlIDs(false)))
	}

	return store
}

func (s *standardImplStore) LookupStandardImplementation(standardID string) (StandardImplementation, error) {
	impl, ok := s.standardImplsByID[standardID]
	if !ok {
		return StandardImplementation{}, fmt.Errorf("invalid standard id %q", standardID)
	}
	return impl, nil
}

func (s *standardImplStore) ListStandardImplementations() []StandardImplementation {
	result := make([]StandardImplementation, 0, len(s.standardImplsByID))
	for _, stdImpl := range s.standardImplsByID {
		result = append(result, stdImpl)
	}
	return result
}

func getChecksForStandard(standard *standards.Standard) []framework.Check {
	var checks []framework.Check
	for _, controlID := range standard.AllControlIDs(true) {
		check := framework.RegistrySingleton().Lookup(controlID)
		if check != nil {
			checks = append(checks, check)
		}
	}
	return checks
}

// DefaultStandardImplementationStore returns the default instance of the standard implementation store, containing all
// built-in standards with their respective checks.
func DefaultStandardImplementationStore() StandardImplementationStore {
	defaultStandardImplStoreInstanceInit.Do(func() {
		defaultStandardImplStoreInstance = createDefaultStandardImplStore()
	})
	return defaultStandardImplStoreInstance
}

//go:generate mockgen-wrapper StandardImplementationStore
