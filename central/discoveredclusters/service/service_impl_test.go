package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestDiscoveredClustersQueryBuilder(t *testing.T) {
	t.Parallel()
	filter := &v1.DiscoveredClustersFilter{
		Names: []string{"my-cluster"},
		Types: []v1.DiscoveredCluster_Metadata_Type{
			v1.DiscoveredCluster_Metadata_EKS,
			v1.DiscoveredCluster_Metadata_GKE,
		},
		Statuses: []v1.DiscoveredCluster_Status{v1.DiscoveredCluster_STATUS_UNSECURED},
	}
	queryBuilder := getQueryBuilderFromFilter(filter)

	rawQuery, err := queryBuilder.RawQuery()
	require.NoError(t, err)

	assert.Contains(t, rawQuery, `Cluster:"my-cluster"`)
	assert.Contains(t, rawQuery, `Cluster Type:"EKS","GKE"`)
	assert.Contains(t, rawQuery, `Cluster Status:"STATUS_UNSECURED"`)
}

func TestDiscoveredClustersQueryBuilderNilFilter(t *testing.T) {
	t.Parallel()
	queryBuilder := getQueryBuilderFromFilter(nil)

	rawQuery, err := queryBuilder.RawQuery()
	require.NoError(t, err)

	assert.Empty(t, rawQuery)
}
