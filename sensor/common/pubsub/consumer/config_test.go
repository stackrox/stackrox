package consumer

import (
	"testing"

	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpec_ToNewConsumer(t *testing.T) {
	t.Run("nil spec returns default consumer", func(t *testing.T) {
		var spec *Spec
		newConsumer, err := spec.ToNewConsumer()
		require.NoError(t, err)
		require.NotNil(t, newConsumer)
	})

	t.Run("default consumer type", func(t *testing.T) {
		spec := &Spec{Type: TypeDefault}
		newConsumer, err := spec.ToNewConsumer()
		require.NoError(t, err)
		require.NotNil(t, newConsumer)
	})

	t.Run("empty consumer type defaults to default", func(t *testing.T) {
		spec := &Spec{Type: ""}
		newConsumer, err := spec.ToNewConsumer()
		require.NoError(t, err)
		require.NotNil(t, newConsumer)
	})

	t.Run("buffered consumer without size", func(t *testing.T) {
		spec := &Spec{Type: TypeBuffered}
		newConsumer, err := spec.ToNewConsumer()
		require.NoError(t, err)
		require.NotNil(t, newConsumer)
	})

	t.Run("buffered consumer with size", func(t *testing.T) {
		spec := &Spec{
			Type: TypeBuffered,
			Size: pointers.Int(100),
		}
		newConsumer, err := spec.ToNewConsumer()
		require.NoError(t, err)
		require.NotNil(t, newConsumer)
	})

	t.Run("invalid consumer type returns error", func(t *testing.T) {
		spec := &Spec{Type: "invalid"}
		newConsumer, err := spec.ToNewConsumer()
		assert.Error(t, err)
		assert.Nil(t, newConsumer)
		assert.Contains(t, err.Error(), "unknown consumer type")
	})
}
