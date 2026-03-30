package manager

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/admission-control/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_GetClusterLabels(t *testing.T) {
	ctx := context.Background()

	t.Run("nil cluster labels returns nil", func(t *testing.T) {
		m := &manager{}
		labels, err := m.GetClusterLabels(ctx, "cluster-id")
		require.NoError(t, err)
		assert.Nil(t, labels)
	})

	t.Run("returns cluster labels", func(t *testing.T) {
		m := &manager{}
		clusterLabels := map[string]string{
			"env":    "prod",
			"region": "us-east-1",
		}
		m.clusterLabels.Store(&clusterLabels)
		labels, err := m.GetClusterLabels(ctx, "cluster-id")
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"env":    "prod",
			"region": "us-east-1",
		}, labels)
	})
}

func TestManager_GetNamespaceLabels(t *testing.T) {
	ctx := context.Background()

	t.Run("returns labels from namespace store", func(t *testing.T) {
		depStore := resources.NewDeploymentStore(nil)
		podStore := resources.NewPodStore()
		nsStore := resources.NewNamespaceStore(depStore, podStore)

		m := &manager{
			namespaces: nsStore,
		}

		// Add namespace to store
		ns := &storage.NamespaceMetadata{
			Name: "test-namespace",
			Labels: map[string]string{
				"team": "backend",
				"tier": "app",
			},
		}
		nsStore.ProcessEvent(central.ResourceAction_CREATE_RESOURCE, ns)

		labels, err := m.GetNamespaceLabels(ctx, "cluster-id", "test-namespace")
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"team": "backend",
			"tier": "app",
		}, labels)
	})

	t.Run("returns nil for non-existent namespace", func(t *testing.T) {
		depStore := resources.NewDeploymentStore(nil)
		podStore := resources.NewPodStore()
		nsStore := resources.NewNamespaceStore(depStore, podStore)

		m := &manager{
			namespaces: nsStore,
		}

		labels, err := m.GetNamespaceLabels(ctx, "cluster-id", "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, labels)
	})
}
