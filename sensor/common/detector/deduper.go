package detector

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// deduper evaluates if a run of detection is needed
type deduper struct {
	hash     map[string]uint64
	hashLock sync.RWMutex
}

func newDeduper() *deduper {
	return &deduper{
		hash: make(map[string]uint64),
	}
}

func (d *deduper) reset() {
	d.hashLock.Lock()
	defer d.hashLock.Unlock()

	d.hash = make(map[string]uint64)
}

func (d *deduper) addDeployment(deployment *storage.Deployment) {
	d.hashLock.Lock()
	defer d.hashLock.Unlock()

	d.hash[deployment.GetId()] = deployment.GetHash()
}

func (d *deduper) needsProcessing(deployment *storage.Deployment) bool {
	hashValue := deployment.GetHash()

	noUpdate := concurrency.WithRLock1(&d.hashLock, func() bool {
		oldHashValue, exists := d.hash[deployment.GetId()]
		return exists && hashValue == oldHashValue
	})
	if noUpdate {
		return false
	}

	d.hashLock.Lock()
	defer d.hashLock.Unlock()

	d.hash[deployment.GetId()] = hashValue
	return true
}

func (d *deduper) removeDeployment(id string) {
	d.hashLock.Lock()
	defer d.hashLock.Unlock()

	delete(d.hash, id)
}
