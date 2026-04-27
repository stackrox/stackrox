package filtercompilers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestSkipContainerTypeToContainerType(t *testing.T) {
	assert.Equal(t, storage.ContainerType_INIT, skipContainerTypeToContainerType(storage.SkipContainerType_SKIP_INIT))
}

func TestBuildContainerSkipSet(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		assert.Nil(t, buildContainerSkipSet(nil))
		assert.Nil(t, buildContainerSkipSet([]storage.SkipContainerType{}))
	})
	t.Run("SKIP_INIT includes INIT", func(t *testing.T) {
		set := buildContainerSkipSet([]storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT})
		_, ok := set[storage.ContainerType_INIT]
		assert.True(t, ok)
		_, ok = set[storage.ContainerType_REGULAR]
		assert.False(t, ok)
	})
}
