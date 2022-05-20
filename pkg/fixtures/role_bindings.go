package fixtures

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

// GetScopedK8SRoleBinding returns a mock K8SRoleBinding belonging to the input scope.
func GetScopedK8SRoleBinding(t *testing.T, id string, clusterID string, namespace string) *storage.K8SRoleBinding {
	roleBinding := &storage.K8SRoleBinding{}
	require.NoError(t, testutils.FullInit(roleBinding, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	roleBinding.ClusterId = clusterID
	roleBinding.Namespace = namespace
	roleBinding.Id = id
	return roleBinding
}
