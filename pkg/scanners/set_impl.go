package scanners

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/sync"
)

type setImpl struct {
	lock sync.RWMutex

	factory      Factory
	integrations map[string]types.ImageScannerWithDataSource
}

var (
	// scannerSortPriority determines the order in which to sort scanners
	// based on scanner type. Types not included will default to 0 and
	// be sorted to the top.
	scannerSortPriority = map[string]int{
		types.Clair:     1,
		types.ScannerV4: 2,
		types.Clairify:  3,
	}
)

// GetAll returns the set of integrations that are active.
func (e *setImpl) GetAll() []types.ImageScannerWithDataSource {
	e.lock.RLock()
	defer e.lock.RUnlock()

	integrations := make([]types.ImageScannerWithDataSource, 0, len(e.integrations))
	for _, i := range e.integrations {
		integrations = append(integrations, i)
	}
	sort.Slice(integrations, func(i, j int) bool {
		iType := integrations[i].GetScanner().Type()
		jType := integrations[j].GetScanner().Type()

		return scannerSortPriority[iType] < scannerSortPriority[jType]
	})
	return integrations
}

// IsEmpty returns whether the set is empty.
func (e *setImpl) IsEmpty() bool {
	e.lock.RLock()
	defer e.lock.RUnlock()

	return len(e.integrations) == 0
}

// Clear removes all present integrations.
func (e *setImpl) Clear() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.integrations = make(map[string]types.ImageScannerWithDataSource)
}

// UpdateImageIntegration updates the integration with the matching id to a new configuration.
func (e *setImpl) UpdateImageIntegration(integration *storage.ImageIntegration) (err error) {
	i, err := e.factory.CreateScanner(integration)
	if err != nil {
		return err
	}

	e.lock.Lock()
	defer e.lock.Unlock()
	e.integrations[integration.GetId()] = i
	return
}

// RemoveImageIntegration removes the integration with a matching id if one exists.
func (e *setImpl) RemoveImageIntegration(id string) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.integrations, id)
	return nil
}
