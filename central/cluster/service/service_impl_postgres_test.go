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

func assertClustersAre(t *testing.T, clusters []*storage.Cluster, expectedNames ...string) {
	t.Helper()
	names := make([]string, len(clusters))
	for i, c := range clusters {
		names[i] = c.GetName()
	}
	require.ElementsMatch(t, expectedNames, names)
}

func TestGetClustersSkewFilteringPostgres(t *testing.T) {
	testutils.SetMainVersion(t, "4.5.0-testing")

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
		{name: "a-matched", sensorVersion: "4.5.9", labels: map[string]string{"name": "a-matched", "env": "prod"}},
		{name: "b-matched", sensorVersion: "4.5.0", labels: map[string]string{"name": "b-matched", "env": "staging"}},
		{name: "c-behind", sensorVersion: "4.4.0", labels: map[string]string{"name": "c-behind", "env": "prod"}},
		{name: "d-ahead", sensorVersion: "4.6.0", labels: map[string]string{"name": "d-ahead", "env": "prod"}},
		{name: "e-incompatible-behind", sensorVersion: "1.0.0", labels: map[string]string{"name": "e-incompatible-behind", "env": "prod"}},
		{name: "f-incompatible-ahead", sensorVersion: "5.5.0", labels: map[string]string{"name": "f-incompatible-ahead", "env": "prod"}},
		{name: "g-unknown", sensorVersion: "", labels: map[string]string{"name": "g-unknown", "env": "prod"}},
	}

	for _, f := range fixtures {
		// MainImage uses a fixed tag because the sensor version is set
		// separately via UpdateClusterStatus, which is what the datastore
		// uses for compatibility classification.
		cluster := &storage.Cluster{
			Name:               f.name,
			MainImage:          "docker.io/stackrox/main:latest",
			CentralApiEndpoint: "central.stackrox:443",
			Labels:             f.labels,
		}
		id, err := ds.AddCluster(ctx, cluster)
		require.NoError(t, err)
		if f.sensorVersion != "" {
			require.NoError(t, ds.UpdateClusterStatus(ctx, id, &storage.ClusterStatus{
				SensorVersion: f.sensorVersion,
			}))
		}
	}

	svc := New(ds, nil, nil, nil)

	t.Run("filter by compatible behind", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		assertClustersAre(t, resp.GetClusters(), "c-behind")
	})

	t.Run("filter by compatible ahead", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		assertClustersAre(t, resp.GetClusters(), "d-ahead")
	})

	t.Run("filter by matched returns multiple", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_MATCHED.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		assertClustersAre(t, resp.GetClusters(), "a-matched", "b-matched")
	})

	t.Run("filter by incompatible behind", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		assertClustersAre(t, resp.GetClusters(), "e-incompatible-behind")
	})

	t.Run("filter by incompatible ahead", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		assertClustersAre(t, resp.GetClusters(), "f-incompatible-ahead")
	})

	t.Run("filter by unknown", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_UNKNOWN.String())

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		assertClustersAre(t, resp.GetClusters(), "g-unknown")
	})

	t.Run("db filter narrows results before skew filter", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s+%s:%s", search.SensorVersionCompatibility,
			storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_MATCHED.String(),
			search.ClusterLabel, "env=prod")

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		assertClustersAre(t, resp.GetClusters(), "a-matched")
	})

	t.Run("no filter returns all clusters", func(t *testing.T) {
		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{
			Query: search.EmptyQuery().String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.GetClusters(), 7)
	})

	t.Run("db filter only without skew filter", func(t *testing.T) {
		query := fmt.Sprintf("%s:%s", search.ClusterLabel, "env=staging")

		resp, err := svc.GetClusters(ctx, &v1.GetClustersRequest{Query: query})
		require.NoError(t, err)
		assertClustersAre(t, resp.GetClusters(), "b-matched")
	})
}
