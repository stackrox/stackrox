package lane

import (
	"testing"

	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/consumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testLaneID pubsub.LaneID = 999
)

func TestSpec_ToConfig(t *testing.T) {
	t.Run("blocking lane without options", func(t *testing.T) {
		spec := Spec{
			ID:   testLaneID,
			Type: TypeBlocking,
		}
		config, err := spec.ToConfig()
		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, testLaneID, config.LaneID())
	})

	t.Run("concurrent lane without options", func(t *testing.T) {
		spec := Spec{
			ID:   testLaneID,
			Type: TypeConcurrent,
		}
		config, err := spec.ToConfig()
		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, testLaneID, config.LaneID())
	})

	t.Run("blocking lane with size", func(t *testing.T) {
		spec := Spec{
			ID:   testLaneID,
			Type: TypeBlocking,
			Size: pointers.Int(50),
		}
		config, err := spec.ToConfig()
		require.NoError(t, err)
		require.NotNil(t, config)
	})

	t.Run("concurrent lane with consumer spec", func(t *testing.T) {
		spec := Spec{
			ID:   testLaneID,
			Type: TypeConcurrent,
			Consumer: &consumer.Spec{
				Type: consumer.TypeBuffered,
				Size: pointers.Int(200),
			},
		}
		config, err := spec.ToConfig()
		require.NoError(t, err)
		require.NotNil(t, config)
	})

	t.Run("invalid lane type returns error", func(t *testing.T) {
		spec := Spec{
			ID:   testLaneID,
			Type: "invalid",
		}
		config, err := spec.ToConfig()
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "unknown lane type")
	})

	t.Run("invalid consumer spec returns error", func(t *testing.T) {
		spec := Spec{
			ID:   testLaneID,
			Type: TypeBlocking,
			Consumer: &consumer.Spec{
				Type: "invalid",
			},
		}
		config, err := spec.ToConfig()
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "invalid consumer spec")
	})
}

func TestSpecsToConfigs(t *testing.T) {
	t.Run("empty specs", func(t *testing.T) {
		configs, err := SpecsToConfigs([]Spec{})
		require.NoError(t, err)
		assert.Empty(t, configs)
	})

	t.Run("multiple valid specs", func(t *testing.T) {
		specs := []Spec{
			{ID: pubsub.LaneID(1), Type: TypeBlocking},
			{ID: pubsub.LaneID(2), Type: TypeConcurrent},
			{ID: pubsub.LaneID(3), Type: TypeBlocking, Size: pointers.Int(100)},
		}
		configs, err := SpecsToConfigs(specs)
		require.NoError(t, err)
		require.Len(t, configs, 3)
		assert.Equal(t, pubsub.LaneID(1), configs[0].LaneID())
		assert.Equal(t, pubsub.LaneID(2), configs[1].LaneID())
		assert.Equal(t, pubsub.LaneID(3), configs[2].LaneID())
	})

	t.Run("invalid spec returns error with index", func(t *testing.T) {
		specs := []Spec{
			{ID: pubsub.LaneID(1), Type: TypeBlocking},
			{ID: pubsub.LaneID(2), Type: "invalid"},
			{ID: pubsub.LaneID(3), Type: TypeConcurrent},
		}
		configs, err := SpecsToConfigs(specs)
		assert.Error(t, err)
		assert.Nil(t, configs)
		assert.Contains(t, err.Error(), "invalid lane spec at index 1")
	})

	t.Run("invalid consumer spec returns error", func(t *testing.T) {
		specs := []Spec{
			{ID: pubsub.LaneID(1), Type: TypeBlocking},
			{
				ID:   pubsub.LaneID(2),
				Type: TypeConcurrent,
				Consumer: &consumer.Spec{
					Type: "invalid-consumer",
				},
			},
		}
		configs, err := SpecsToConfigs(specs)
		assert.Error(t, err)
		assert.Nil(t, configs)
		assert.Contains(t, err.Error(), "invalid lane spec at index 1")
		assert.Contains(t, err.Error(), "invalid consumer spec")
	})
}
