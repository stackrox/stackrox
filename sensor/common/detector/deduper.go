package detector

import (
	"github.com/mitchellh/hashstructure"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/sync"
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
	hashValue, err := hashstructure.Hash(deployment, &hashstructure.HashOptions{})
	if err != nil {
		log.Errorf("error calculating hash of deployment %q: %v", deployment.GetName(), err)
		return
	}

	d.hashLock.Lock()
	defer d.hashLock.Unlock()

	d.hash[deployment.GetId()] = hashValue
}

func (d *deduper) needsProcessing(deployment *storage.Deployment) bool {
	// if removal then remove from hash and send empty alerts
	hashValue, err := hashstructure.Hash(deployment, &hashstructure.HashOptions{})
	if err != nil {
		log.Errorf("error calculating hash of deployment %q: %v", deployment.GetName(), err)
		return true
	}

	var noUpdate bool
	concurrency.WithRLock(&d.hashLock, func() {
		oldHashValue, exists := d.hash[deployment.GetId()]
		noUpdate = exists && hashValue == oldHashValue
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
