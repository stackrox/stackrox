package aggregator

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
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
	if !features.ActiveVulnMgmt.Enabled() {
		return
	}
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, indicator := range indicators {
		// Guard against invalid indicators
		if indicator.GetImageId() == "" || indicator.GetContainerName() == "" || indicator.GetDeploymentId() == "" {
			log.Debugf("invalid indicator with imageID %q, containerName %q, and deploymentID %q", indicator.GetImageId(), indicator.GetContainerName(), indicator.GetDeploymentId())
			continue
		}

		containerMap, ok := a.cache[indicator.GetDeploymentId()]
		if !ok {
			containerMap = make(map[string]*ProcessUpdate)
			a.cache[indicator.GetDeploymentId()] = containerMap
		}

		update, ok := containerMap[indicator.GetContainerName()]
		if !ok {
			update = &ProcessUpdate{
				ContainerName: indicator.GetContainerName(),
				NewPaths:      set.NewStringSet(),
				ImageID:       indicator.GetImageId(),
				state:         FromDatabase,
			}
			containerMap[indicator.GetContainerName()] = update
		}

		// Skip indicators from a different image
		if indicator.GetImageId() != update.ImageID {
			log.Debugf("skip unexpected image %s in indicator, expecting image %s for deployment %s, container %s", indicator.GetImageId(), update.ImageID, indicator.GetDeploymentId(), update.ContainerName)
			continue
		}

		if update.FromCache() {
			update.NewPaths.Add(indicator.GetSignal().GetExecFilePath())
		}
	}
}

// GetAndPrune gets the deployments and their updates to process.
func (a *aggregatorImpl) GetAndPrune(imageScanned func(string) bool, deploymentsSet set.StringSet) map[string][]*ProcessUpdate {
	if !features.ActiveVulnMgmt.Enabled() {
		return nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()

	updates := make(map[string][]*ProcessUpdate)
	for deploymentID, containerMap := range a.cache {
		if !deploymentsSet.Contains(deploymentID) {
			delete(a.cache, deploymentID)
			continue
		}
		for containerName, update := range containerMap {
			// No new process executed or the image is not scanned yet
			if (update.FromCache() && len(update.NewPaths) == 0) ||
				(!update.ToBeRemoved() && !imageScanned(update.ImageID)) {
				continue
			}

			// Collect this update
			updates[deploymentID] = append(updates[deploymentID], update)

			if update.ToBeRemoved() {
				delete(containerMap, containerName)
				continue
			}

			state := update.state
			if update.FromDatabase() {
				state = FromCache
			}
			// Create new update and detach old one
			containerMap[containerName] = &ProcessUpdate{
				ContainerName: containerName,
				ImageID:       update.ImageID,
				NewPaths:      set.NewStringSet(),
				state:         state,
			}
		}
	}
	return updates
}

// RefreshDeployment maintains cache with current deployment
func (a *aggregatorImpl) RefreshDeployment(deployment *storage.Deployment) {
	if !features.ActiveVulnMgmt.Enabled() {
		return
	}
	a.lock.Lock()
	defer a.lock.Unlock()

	containerMap, ok := a.cache[deployment.GetId()]
	if !ok {
		containerMap = make(map[string]*ProcessUpdate)
		a.cache[deployment.GetId()] = containerMap
	}

	containerNames := set.NewStringSet()
	for _, container := range deployment.GetContainers() {
		containerNames.Add(container.Name)
		update, ok := containerMap[container.GetName()]
		if !ok {
			update = &ProcessUpdate{
				ContainerName: container.GetName(),
				NewPaths:      set.NewStringSet(),
				ImageID:       container.GetImage().GetId(),
				state:         FromDatabase,
			}
			containerMap[container.GetName()] = update
		}
		// On container image change, we invalidate existing active components and calculate again.
		if update.ImageID != container.GetImage().GetId() {
			update.ImageID = container.GetImage().GetId()
			update.NewPaths.Clear()
			update.state = FromDatabase
		}
	}

	// Delete updates for removed containers
	for containerName, update := range containerMap {
		if !containerNames.Contains(containerName) {
			update.state = ToBeRemoved
			update.ImageID = ""
			update.NewPaths.Clear()
		}
	}
}
