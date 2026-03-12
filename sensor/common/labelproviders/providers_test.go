package labelproviders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/sensor/common/clusterlabels"
	"github.com/stretchr/testify/assert"
)

func TestClusterLabelProviderAdapter(t *testing.T) {
	tests := []struct {
		name           string
		labels         map[string]string
		clusterID      string
		expectedLabels map[string]string
	}{
		{
			name:           "returns labels from store",
			labels:         map[string]string{"env": "prod", "team": "platform"},
			clusterID:      "cluster-1",
			expectedLabels: map[string]string{"env": "prod", "team": "platform"},
		},
		{
			name:           "cluster ID is ignored",
			labels:         map[string]string{"region": "us-west"},
			clusterID:      "any-id",
			expectedLabels: map[string]string{"region": "us-west"},
		},
		{
			name:           "empty labels",
			labels:         map[string]string{},
			clusterID:      "cluster-1",
			expectedLabels: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := clusterlabels.NewStore()
			store.Set(tt.labels)
			provider := NewClusterLabelProviderAdapter(store)

			result, err := provider.GetClusterLabels(context.Background(), tt.clusterID)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedLabels, result)
		})
	}
}

func TestNamespaceLabelProviderAdapter(t *testing.T) {
	tests := []struct {
		name           string
		namespaceID    string
		expectedLabels map[string]string
		expectedFound  bool
	}{
		{
			name:           "namespace found",
			namespaceID:    "ns-123",
			expectedLabels: map[string]string{"app": "test", "tier": "backend"},
			expectedFound:  true,
		},
		{
			name:           "namespace not found",
			namespaceID:    "ns-456",
			expectedLabels: nil,
			expectedFound:  false,
		},
	}

	lookupFunc := func(id string) (map[string]string, bool) {
		if id == "ns-123" {
			return map[string]string{"app": "test", "tier": "backend"}, true
		}
		return nil, false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewNamespaceLabelProviderAdapter(lookupFunc)

			result, err := provider.GetNamespaceLabels(context.Background(), tt.namespaceID)
			assert.NoError(t, err)
			if tt.expectedFound {
				assert.Equal(t, tt.expectedLabels, result)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}
