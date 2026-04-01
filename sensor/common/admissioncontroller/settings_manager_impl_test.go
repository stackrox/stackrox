package admissioncontroller

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	storeMocks "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type mockClusterLabelsGetter struct {
	labels map[string]string
}

func (m *mockClusterLabelsGetter) Get() map[string]string {
	return m.labels
}

type mockNamespaceGetter struct {
	namespaces []*storage.NamespaceMetadata
}

func (m *mockNamespaceGetter) GetAll() []*storage.NamespaceMetadata {
	return m.namespaces
}

type mockClusterIDWaiter struct {
	id string
}

func (m *mockClusterIDWaiter) Get() string {
	return m.id
}

func TestSettingsManager_GetResourcesForSync(t *testing.T) {
	t.Run("includes namespaces in sync", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		clusterID := &mockClusterIDWaiter{id: "cluster-123"}
		clusterLabels := &mockClusterLabelsGetter{}
		deployments := storeMocks.NewMockDeploymentStore(ctrl)
		pods := storeMocks.NewMockPodStore(ctrl)

		deployments.EXPECT().GetAll().Return(nil).AnyTimes()
		pods.EXPECT().GetAll().Return(nil).AnyTimes()
		namespaces := &mockNamespaceGetter{
			namespaces: []*storage.NamespaceMetadata{
				{
					Id:   "ns-1",
					Name: "test-namespace-1",
					Labels: map[string]string{
						"team": "backend",
					},
				},
				{
					Id:   "ns-2",
					Name: "test-namespace-2",
					Labels: map[string]string{
						"team": "frontend",
					},
				},
			},
		}

		mgr := NewSettingsManager(clusterID, clusterLabels, deployments, pods, namespaces).(*settingsManager)

		resources := mgr.GetResourcesForSync()

		// Should have 2 namespace resources
		var namespaceResources int
		for _, res := range resources {
			if res.GetNamespace() != nil {
				namespaceResources++
			}
		}
		assert.Equal(t, 2, namespaceResources, "should sync all namespaces")
	})

	t.Run("handles nil namespaces", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		clusterID := &mockClusterIDWaiter{id: "cluster-123"}
		clusterLabels := &mockClusterLabelsGetter{}
		deployments := storeMocks.NewMockDeploymentStore(ctrl)
		pods := storeMocks.NewMockPodStore(ctrl)

		deployments.EXPECT().GetAll().Return(nil).AnyTimes()
		pods.EXPECT().GetAll().Return(nil).AnyTimes()

		mgr := NewSettingsManager(clusterID, clusterLabels, deployments, pods, nil).(*settingsManager)

		resources := mgr.GetResourcesForSync()

		// Should not panic and should not include namespaces
		var namespaceResources int
		for _, res := range resources {
			if res.GetNamespace() != nil {
				namespaceResources++
			}
		}
		assert.Equal(t, 0, namespaceResources, "should not include namespaces when nil")
	})
}

func TestSettingsManager_UpdateResources_NamespaceEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	clusterID := &mockClusterIDWaiter{id: "cluster-123"}
	clusterLabels := &mockClusterLabelsGetter{}
	deployments := storeMocks.NewMockDeploymentStore(ctrl)
	pods := storeMocks.NewMockPodStore(ctrl)
	namespaces := &mockNamespaceGetter{}

	mgr := NewSettingsManager(clusterID, clusterLabels, deployments, pods, namespaces).(*settingsManager)

	// Test that all namespace event types are forwarded (not just deletes)
	testCases := []struct {
		name   string
		action central.ResourceAction
	}{
		{
			name:   "CREATE namespace event",
			action: central.ResourceAction_CREATE_RESOURCE,
		},
		{
			name:   "UPDATE namespace event",
			action: central.ResourceAction_UPDATE_RESOURCE,
		},
		{
			name:   "REMOVE namespace event",
			action: central.ResourceAction_REMOVE_RESOURCE,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nsEvent := &central.SensorEvent{
				Action: tc.action,
				Resource: &central.SensorEvent_Namespace{
					Namespace: &storage.NamespaceMetadata{
						Id:   "ns-123",
						Name: "test-namespace",
					},
				},
			}

			// Should not panic - UpdateResources forwards all namespace events
			assert.NotPanics(t, func() {
				mgr.UpdateResources(nsEvent)
			})
		})
	}
}
