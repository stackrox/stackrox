package aggregator

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

type updateState int

const (
	// FromDatabase is the state to load indicators from database
	FromDatabase updateState = iota
	// FromCache is the state to get aggregated paths from cache
	FromCache
	// ToBeRemoved is the state to remove a container name from all existing active components
	ToBeRemoved
)

var (
	log = logging.LoggerForModule()
)

// ProcessUpdate holds all state changes and processes executed since last prune
type ProcessUpdate struct {
	ImageID       string
	ContainerName string
	NewPaths      set.StringSet

	state updateState
}

// NewProcessUpdate creates a process update
func NewProcessUpdate(imageID, containerName string, newPaths set.StringSet, state updateState) *ProcessUpdate {
	return &ProcessUpdate{
		ImageID:       imageID,
		ContainerName: containerName,
		NewPaths:      newPaths,
		state:         state,
	}
}

// FromDatabase returns whether this update is to load process indicators from the database
func (u *ProcessUpdate) FromDatabase() bool {
	return u.state == FromDatabase
}

// FromCache returns whether this update is to load executable paths from itself
func (u *ProcessUpdate) FromCache() bool {
	return u.state == FromCache
}

// ToBeRemoved returns whether this update is to remove a container from all existing active components
func (u *ProcessUpdate) ToBeRemoved() bool {
	return u.state == ToBeRemoved
}

type aggregatorImpl struct {
	lock sync.Mutex
	// Map from deployment id to map from container names to updates
	cache map[string]map[string]*ProcessUpdate
}

// Add adds indicators to the cache if applicable
func (a *aggregatorImpl) Add(indicators []*storage.ProcessIndicator) {
	return
}

// GetAndPrune gets the deployments and their updates to process.
func (a *aggregatorImpl) GetAndPrune(imageScanned func(string) bool, deploymentsSet set.StringSet) map[string][]*ProcessUpdate {
	return nil
}

// RefreshDeployment maintains cache with current deployment
func (a *aggregatorImpl) RefreshDeployment(deployment *storage.Deployment) {
	return
}
