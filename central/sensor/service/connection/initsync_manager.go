package connection

import (
	"fmt"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

type initSyncManager struct {
	mutex sync.Mutex

	maxSensors int
	sensors    set.StringSet
}

// NewInitSyncManager creates an initSyncManager with max sensors
// retrieved from env variable, ensuring it is non-negative.
func NewInitSyncManager() *initSyncManager {
	maxSensors := env.CentralMaxInitSyncSensors.IntegerSetting()
	if maxSensors < 0 {
		panic(fmt.Sprintf("Negative number is not allowed for max init sync sensors. Check env variable: %q", env.CentralMaxInitSyncSensors.EnvVar()))
	}

	return &initSyncManager{
		maxSensors: maxSensors,
		sensors:    set.NewStringSet(),
	}
}

func (m *initSyncManager) Add(clusterID string) bool {
	if m.maxSensors == 0 {
		return true
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.sensors) >= m.maxSensors {
		return false
	}
	m.sensors.Add(clusterID)

	return true
}

func (m *initSyncManager) Remove(clusterID string) {
	if m.maxSensors == 0 {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.sensors.Remove(clusterID)
}
