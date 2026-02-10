package config

import (
	"testing"

	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testLaneID pubsub.LaneID = 999
)

func TestConsumerSpec_ToNewConsumer(t *testing.T) {
	t.Run("nil spec returns default consumer", func(t *testing.T) {
		var spec *ConsumerSpec
		newConsumer, err := spec.ToNewConsumer()
		require.NoError(t, err)
		require.NotNil(t, newConsumer)
	})

	t.Run("default consumer type", func(t *testing.T) {
		spec := &ConsumerSpec{Type: ConsumerTypeDefault}
		newConsumer, err := spec.ToNewConsumer()
		require.NoError(t, err)
		require.NotNil(t, newConsumer)
	})

	t.Run("empty consumer type defaults to default", func(t *testing.T) {
		spec := &ConsumerSpec{Type: ""}
		newConsumer, err := spec.ToNewConsumer()
		require.NoError(t, err)
		require.NotNil(t, newConsumer)
	})

	t.Run("buffered consumer without size", func(t *testing.T) {
		spec := &ConsumerSpec{Type: ConsumerTypeBuffered}
		newConsumer, err := spec.ToNewConsumer()
		require.NoError(t, err)
		require.NotNil(t, newConsumer)
	})

	t.Run("buffered consumer with size", func(t *testing.T) {
		spec := &ConsumerSpec{
			Type: ConsumerTypeBuffered,
			Size: pointers.Int(100),
		}
		newConsumer, err := spec.ToNewConsumer()
		require.NoError(t, err)
		require.NotNil(t, newConsumer)
	})

	t.Run("invalid consumer type returns error", func(t *testing.T) {
		spec := &ConsumerSpec{Type: "invalid"}
		newConsumer, err := spec.ToNewConsumer()
		assert.Error(t, err)
		assert.Nil(t, newConsumer)
		assert.Contains(t, err.Error(), "unknown consumer type")
	})
}

func TestLaneSpec_ToConfig(t *testing.T) {
	t.Run("blocking lane without options", func(t *testing.T) {
		spec := LaneSpec{
			ID:   testLaneID,
			Type: LaneTypeBlocking,
		}
		config, err := spec.ToConfig()
		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, testLaneID, config.LaneID())
	})

	t.Run("concurrent lane without options", func(t *testing.T) {
		spec := LaneSpec{
			ID:   testLaneID,
			Type: LaneTypeConcurrent,
		}
		config, err := spec.ToConfig()
		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, testLaneID, config.LaneID())
	})

	t.Run("blocking lane with size", func(t *testing.T) {
		spec := LaneSpec{
			ID:   testLaneID,
			Type: LaneTypeBlocking,
			Size: pointers.Int(50),
		}
		config, err := spec.ToConfig()
		require.NoError(t, err)
		require.NotNil(t, config)
	})

	t.Run("concurrent lane with consumer spec", func(t *testing.T) {
		spec := LaneSpec{
			ID:   testLaneID,
			Type: LaneTypeConcurrent,
			Consumer: &ConsumerSpec{
				Type: ConsumerTypeBuffered,
				Size: pointers.Int(200),
			},
		}
		config, err := spec.ToConfig()
		require.NoError(t, err)
		require.NotNil(t, config)
	})

	t.Run("invalid lane type returns error", func(t *testing.T) {
		spec := LaneSpec{
			ID:   testLaneID,
			Type: "invalid",
		}
		config, err := spec.ToConfig()
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "unknown lane type")
	})

	t.Run("invalid consumer spec returns error", func(t *testing.T) {
		spec := LaneSpec{
			ID:   testLaneID,
			Type: LaneTypeBlocking,
			Consumer: &ConsumerSpec{
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
		configs, err := SpecsToConfigs([]LaneSpec{})
		require.NoError(t, err)
		assert.Empty(t, configs)
	})

	t.Run("multiple valid specs", func(t *testing.T) {
		specs := []LaneSpec{
			{ID: pubsub.LaneID(1), Type: LaneTypeBlocking},
			{ID: pubsub.LaneID(2), Type: LaneTypeConcurrent},
			{ID: pubsub.LaneID(3), Type: LaneTypeBlocking, Size: pointers.Int(100)},
		}
		configs, err := SpecsToConfigs(specs)
		require.NoError(t, err)
		require.Len(t, configs, 3)
		assert.Equal(t, pubsub.LaneID(1), configs[0].LaneID())
		assert.Equal(t, pubsub.LaneID(2), configs[1].LaneID())
		assert.Equal(t, pubsub.LaneID(3), configs[2].LaneID())
	})

	t.Run("invalid spec returns error with index", func(t *testing.T) {
		specs := []LaneSpec{
			{ID: pubsub.LaneID(1), Type: LaneTypeBlocking},
			{ID: pubsub.LaneID(2), Type: "invalid"},
			{ID: pubsub.LaneID(3), Type: LaneTypeConcurrent},
		}
		configs, err := SpecsToConfigs(specs)
		assert.Error(t, err)
		assert.Nil(t, configs)
		assert.Contains(t, err.Error(), "invalid lane spec at index 1")
	})

	t.Run("invalid consumer spec returns error", func(t *testing.T) {
		specs := []LaneSpec{
			{ID: pubsub.LaneID(1), Type: LaneTypeBlocking},
			{
				ID:   pubsub.LaneID(2),
				Type: LaneTypeConcurrent,
				Consumer: &ConsumerSpec{
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
