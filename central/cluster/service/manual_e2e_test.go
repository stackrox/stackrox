//go:build sql_integration

// THROWAWAY / MANUAL TEST -- not meant to be committed.
//
// Exercises the real service.serviceImpl.GetClusters implementation (query
// splitting, in-memory skew filtering, in-memory pagination) against a real
// Postgres-backed cluster datastore, so we can see the whole
// service -> datastore -> Postgres path behave end to end without needing a
// fully deployed Central or a real Sensor.
//
// Requires a local Postgres reachable on port 5432, e.g.:
//
//	docker run --rm --env POSTGRES_USER="$USER" \
//	  --env POSTGRES_HOST_AUTH_METHOD=trust --publish 5432:5432 \
//	  docker.io/library/postgres:15
//
// Run with:
//
//	go test -tags sql_integration -v -run TestManualE2EGetClusters \
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

func TestManualE2EGetClusters(t *testing.T) {
	// Pin Central's own version so the sensor-version-compatibility
	// classification below is deterministic.
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
		{name: "a-matched", sensorVersion: "4.5.9", labels: map[string]string{"name": "a-matched", "major": "4"}},           // MATCHED
		{name: "b-matched", sensorVersion: "4.5.0", labels: map[string]string{"name": "b-matched", "major": "4"}},           // MATCHED
		{name: "c-behind", sensorVersion: "4.4.0", labels: map[string]string{"name": "c-behind", "major": "4"}},             // COMPATIBLE_BEHIND
		{name: "d-ahead", sensorVersion: "4.6.0", labels: map[string]string{"name": "d-ahead", "major": "4"}},               // COMPATIBLE_AHEAD
		{name: "e-incompatible", sensorVersion: "1.0.0", labels: map[string]string{"name": "e-incompatible", "major": "1"}}, // INCOMPATIBLE_BEHIND
		{name: "f-incompatible", sensorVersion: "5.5.0", labels: map[string]string{"name": "f-incompatible", "major": "5"}}, // INCOMPATIBLE_AHEAD
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

	t.Run("skew filter only", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		require.Len(t, resp.GetClusters(), 1)
		require.Equal(t, "c-behind", resp.GetClusters()[0].GetName())
	})

	t.Run("pagination only", func(t *testing.T) {
		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{
			Query: search.EmptyQuery().String(),
			Pagination: &v1.Pagination{
				Offset: 1,
				Limit:  2,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.GetClusters(), 2)
		// Clusters come back name-sorted: a-matched, b-matched, c-behind, d-ahead, e-incompatible.
		require.Equal(t, "b-matched", resp.GetClusters()[0].GetName())
		require.Equal(t, "c-behind", resp.GetClusters()[1].GetName())
	})

	t.Run("skew filter + pagination combined", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_MATCHED.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{
			Query: query,
			Pagination: &v1.Pagination{
				Offset: 1,
				Limit:  1,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.GetClusters(), 1)
		require.Equal(t, "b-matched", resp.GetClusters()[0].GetName())
	})
	t.Run("db filter + skew filter", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s+%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_MATCHED.String(),
			search.ClusterLabel, "major=4")

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{
			Query: query,
			Pagination: &v1.Pagination{
				Offset: 1,
				Limit:  1,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.GetClusters(), 1)
		require.Equal(t, "b-matched", resp.GetClusters()[0].GetName())
	})
}
