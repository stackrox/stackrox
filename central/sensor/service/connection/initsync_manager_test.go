package connection

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
)

func TestNewInitSyncManagerNegativeMaxPanics(t *testing.T) {
	t.Setenv(env.CentralMaxInitSyncSensors.EnvVar(), "-1")
	assert.Panics(t, func() { NewInitSyncManager() })
}

func TestInitSyncManagerZeroNoLimit(t *testing.T) {
	t.Setenv(env.CentralMaxInitSyncSensors.EnvVar(), "0")
	m := NewInitSyncManager()
	assert.Equal(t, 0, m.maxSensors)

	assert.True(t, m.Add("test-1"), "Can add if limit is set to 0")
	assert.Len(t, m.sensors, 0)

	m.Remove("test-2")
	assert.Len(t, m.sensors, 0)
}

func TestInitSyncManagerDefault(t *testing.T) {
	m := NewInitSyncManager()
	assert.Equal(t, 0, m.maxSensors)

	assert.True(t, m.Add("test-1"), "Can add if limit is set to 0")
	assert.Len(t, m.sensors, 0)
}

func TestInitSyncManager(t *testing.T) {
	t.Setenv(env.CentralMaxInitSyncSensors.EnvVar(), "3")
	m := NewInitSyncManager()

	for i := 0; i < 3; i++ {
		assert.True(t, m.Add(fmt.Sprintf("test-%d", i)))
	}
	assert.False(t, m.Add("test-a"), "Unable to add after limit is reached")
	assert.Len(t, m.sensors, 3)

	m.Remove("test-a")
	assert.False(t, m.Add("test-a"), "Unable to add after removing non-existing")

	m.Remove("test-1")
	assert.Len(t, m.sensors, 2)
	assert.True(t, m.Add("test-a"), "Can add after one is removed")
	assert.Len(t, m.sensors, 3)

	assert.False(t, m.Add("test-b"), "Unable to add after limit is reached")
}
