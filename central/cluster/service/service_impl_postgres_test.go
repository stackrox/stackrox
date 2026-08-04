//go:build sql_integration

// Exercises the real service.serviceImpl.GetClusters implementation (query
// splitting, in-memory skew filtering) against a real Postgres-backed cluster
// datastore.
//
// Requires a local Postgres reachable on port 5432, e.g.:
//
//	docker run --rm --env POSTGRES_USER="$USER" \
//	  --env POSTGRES_HOST_AUTH_METHOD=trust --publish 5432:5432 \
//	  docker.io/library/postgres:15
//
// Run with:
//
//	go test -tags sql_integration -v -run TestGetClustersSkewFilteringPostgres \
//	  ./central/cluster/service/...
package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/cluster/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/require"
)

func TestGetClustersSkewFilteringPostgres(t *testing.T) {
	testutils.SetMainVersion(t, "4.5.0")

	db := pgtest.ForT(t)
	ds, err := datastore.GetTestPostgresDataStore(t, db)
	require.NoError(t, err)

	ctx := sac.WithAllAccess(context.Background())

	type fixture struct {
		name          string
		sensorVersion string
		labels        map[string]string
	}
	fixtures := []fixture{
		{name: "a-matched", sensorVersion: "4.5.9", labels: map[string]string{"name": "a-matched", "major": "4"}},
		{name: "b-matched", sensorVersion: "4.5.0", labels: map[string]string{"name": "b-matched", "major": "4"}},
		{name: "c-behind", sensorVersion: "4.4.0", labels: map[string]string{"name": "c-behind", "major": "4"}},
		{name: "d-ahead", sensorVersion: "4.6.0", labels: map[string]string{"name": "d-ahead", "major": "4"}},
		{name: "e-incompatible-behind", sensorVersion: "1.0.0", labels: map[string]string{"name": "e-incompatible-behind", "major": "1"}},
		{name: "f-incompatible-ahead", sensorVersion: "5.5.0", labels: map[string]string{"name": "f-incompatible-ahead", "major": "5"}},
	}

	for _, f := range fixtures {
		id, err := ds.AddCluster(ctx, &storage.Cluster{
			Name:               f.name,
			MainImage:          fmt.Sprintf("docker.io/stackrox/main:%s", f.sensorVersion),
			CentralApiEndpoint: "central.stackrox:443",
			Labels:             f.labels,
		})
		require.NoError(t, err)
		require.NoError(t, ds.UpdateClusterStatus(ctx, id, &storage.ClusterStatus{
			SensorVersion: f.sensorVersion,
		}))
	}

	svc := New(ds, nil, nil, nil)

	t.Run("filter by compatible behind", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		require.Len(t, resp.GetClusters(), 1)
		require.Equal(t, "c-behind", resp.GetClusters()[0].GetName())
	})

	t.Run("filter by matched returns multiple", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_MATCHED.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		require.Len(t, resp.GetClusters(), 2)
		names := []string{resp.GetClusters()[0].GetName(), resp.GetClusters()[1].GetName()}
		require.ElementsMatch(t, []string{"a-matched", "b-matched"}, names)
	})

	t.Run("filter by incompatible behind", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		require.Len(t, resp.GetClusters(), 1)
		require.Equal(t, "e-incompatible-behind", resp.GetClusters()[0].GetName())
	})

	t.Run("filter by incompatible ahead", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		require.Len(t, resp.GetClusters(), 1)
		require.Equal(t, "f-incompatible-ahead", resp.GetClusters()[0].GetName())
	})

	t.Run("filter returns empty results", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_UNKNOWN.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		require.Empty(t, resp.GetClusters())
	})

	t.Run("db filter combined with skew filter", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s+%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_MATCHED.String(),
			search.ClusterLabel, "major=4")

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		require.Len(t, resp.GetClusters(), 2)
		names := []string{resp.GetClusters()[0].GetName(), resp.GetClusters()[1].GetName()}
		require.ElementsMatch(t, []string{"a-matched", "b-matched"}, names)
	})
}
