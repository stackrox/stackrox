package fixtures

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

// GetScopedK8SRole returns a mock K8SRole belonging to the input scope.
func GetScopedK8SRole(t *testing.T, id string, clusterID string, namespace string) *storage.K8SRole {
	role := &storage.K8SRole{}
	require.NoError(t, testutils.FullInit(role, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	role.Id = id
	role.ClusterId = clusterID
	role.Namespace = namespace

	return role
}
