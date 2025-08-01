package manager

import (
	"testing"

	"github.com/stackrox/rox/sensor/common/networkflow/updatecomputer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateComputerOptions(t *testing.T) {
	// Test that the manager can be configured with different update computers
	t.Run("WithLegacyUpdateComputer", func(t *testing.T) {
		mgr := &networkFlowManager{}
		mgr.updateComputer = updatecomputer.NewLegacyUpdateComputer()

		require.NotNil(t, mgr.updateComputer)
		_, ok := mgr.updateComputer.(*updatecomputer.LegacyUpdateComputer)
		assert.True(t, ok, "Should use LegacyUpdateComputer")
	})

	t.Run("WithCategorizedUpdateComputer", func(t *testing.T) {
		mgr := &networkFlowManager{}
		mgr.updateComputer = updatecomputer.NewCategorizedUpdateComputer()

		require.NotNil(t, mgr.updateComputer)
		_, ok := mgr.updateComputer.(*updatecomputer.CategorizedUpdateComputer)
		assert.True(t, ok, "Should use CategorizedUpdateComputer")
	})

	t.Run("NewUpdateComputer", func(t *testing.T) {
		t.Run("Legacy", func(t *testing.T) {
			mgr := &networkFlowManager{}
			mgr.updateComputer = updatecomputer.NewUpdateComputer(updatecomputer.LegacyUpdateComputerType)

			require.NotNil(t, mgr.updateComputer)
			_, ok := mgr.updateComputer.(*updatecomputer.LegacyUpdateComputer)
			assert.True(t, ok, "Should use LegacyUpdateComputer")
		})

		t.Run("Categorized", func(t *testing.T) {
			mgr := &networkFlowManager{}
			mgr.updateComputer = updatecomputer.NewUpdateComputer(updatecomputer.CategorizedUpdateComputerType)

			require.NotNil(t, mgr.updateComputer)
			_, ok := mgr.updateComputer.(*updatecomputer.CategorizedUpdateComputer)
			assert.True(t, ok, "Should use CategorizedUpdateComputer")
		})

		t.Run("Unknown defaults to Categorized", func(t *testing.T) {
			mgr := &networkFlowManager{}
			mgr.updateComputer = updatecomputer.NewUpdateComputer("unknown")

			require.NotNil(t, mgr.updateComputer)
			_, ok := mgr.updateComputer.(*updatecomputer.CategorizedUpdateComputer)
			assert.True(t, ok, "Should default to CategorizedUpdateComputer")
		})
	})
}
