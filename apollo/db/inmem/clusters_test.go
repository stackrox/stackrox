package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testClusters(t *testing.T, insertStorage, retrievalStorage db.ClusterStorage) {
	clusters := []*v1.Cluster{
		{
			Name:        "cluster1",
			ApolloImage: "test-dtr.example.com/apollo",
		},
		{
			Name:        "cluster2",
			ApolloImage: "docker.io/stackrox/apollo",
		},
	}

	// Test Add
	for _, b := range clusters {
		assert.NoError(t, insertStorage.AddCluster(b))
	}

	// Test that adding again fails
	for _, b := range clusters {
		assert.Error(t, insertStorage.AddCluster(b))
	}

	for _, b := range clusters {
		got, exists, err := retrievalStorage.GetCluster(b.Name)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, b)
	}

	// Test Update
	for _, b := range clusters {
		b.ApolloImage = b.ApolloImage + "/apollo"
	}

	for _, b := range clusters {
		assert.NoError(t, insertStorage.UpdateCluster(b))
	}

	for _, b := range clusters {
		got, exists, err := retrievalStorage.GetCluster(b.GetName())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, b)
	}

	// Test Remove
	for _, b := range clusters {
		assert.NoError(t, insertStorage.RemoveCluster(b.GetName()))
	}

	for _, b := range clusters {
		_, exists, err := retrievalStorage.GetCluster(b.GetName())
		assert.NoError(t, err)
		assert.False(t, exists)
	}

}

func TestClustersPersistence(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	require.NoError(t, err)
	storage := newClusterStore(persistent)
	testClusters(t, storage, persistent)
}

func TestClusters(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	require.NoError(t, err)
	storage := newClusterStore(persistent)
	testClusters(t, storage, storage)
}

func TestClustersFiltering(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	require.NoError(t, err)
	storage := newClusterStore(persistent)

	clusters := []*v1.Cluster{
		{
			Name:        "cluster1",
			ApolloImage: "test-dtr.example.com/apollo",
		},
		{
			Name:        "cluster2",
			ApolloImage: "docker.io/stackrox/apollo",
		},
	}

	// Test Add
	for _, r := range clusters {
		assert.NoError(t, storage.AddCluster(r))
	}

	actualClusters, err := storage.GetClusters()
	assert.NoError(t, err)
	assert.Equal(t, clusters, actualClusters)
}
