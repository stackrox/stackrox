package centralsensor

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestSecuredClusterIsNotManagedManually(t *testing.T) {
	tests := []struct {
		name            string
		managedBy       storage.ManagerType
		expectedOutcome bool
	}{
		{
			name:            "Unknown manager type",
			managedBy:       storage.ManagerType_MANAGER_TYPE_UNKNOWN,
			expectedOutcome: false,
		},
		{
			name:            "Manually managed",
			managedBy:       storage.ManagerType_MANAGER_TYPE_MANUAL,
			expectedOutcome: false,
		},
		{
			name:            "Managed by Helm",
			managedBy:       storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			expectedOutcome: true,
		},
		{
			name:            "Managed by Operator",
			managedBy:       storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR,
			expectedOutcome: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &central.HelmManagedConfigInit{}
			config.ManagedBy = tt.managedBy

			result := SecuredClusterIsNotManagedManually(config)
			assert.Equal(t, tt.expectedOutcome, result)
		})
	}
}
