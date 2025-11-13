//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

// sensorEventTypeDistribution defines the percentage distribution of sensor event types
type sensorEventTypeDistribution struct {
	name       string
	percentage int
	generator  func(id string) interface{}
}

// generateSensorEventTypes returns a list of sensor event types with their percentages
func generateSensorEventTypes() []sensorEventTypeDistribution {
	return []sensorEventTypeDistribution{
		{
			name:       "Pod",
			percentage: 20,
			generator: func(id string) interface{} {
				return &central.SensorEvent_Pod{
					Pod: &storage.Pod{
						Id:   id,
						Name: fmt.Sprintf("pod-%s", id[:8]),
					},
				}
			},
		},
		{
			name:       "Deployment",
			percentage: 15,
			generator: func(id string) interface{} {
				return &central.SensorEvent_Deployment{
					Deployment: &storage.Deployment{
						Id:   id,
						Name: fmt.Sprintf("deployment-%s", id[:8]),
					},
				}
			},
		},
		{
			name:       "Namespace",
			percentage: 5,
			generator: func(id string) interface{} {
				return &central.SensorEvent_Namespace{
					Namespace: &storage.NamespaceMetadata{
						Id:   id,
						Name: fmt.Sprintf("namespace-%s", id[:8]),
					},
				}
			},
		},
		{
			name:       "AlertResults",
			percentage: 15,
			generator: func(id string) interface{} {
				return &central.SensorEvent_AlertResults{
					AlertResults: &central.AlertResults{
						DeploymentId: id,
						Alerts:       []*storage.Alert{},
					},
				}
			},
		},
		{
			name:       "NetworkPolicy",
			percentage: 5,
			generator: func(id string) interface{} {
				return &central.SensorEvent_NetworkPolicy{
					NetworkPolicy: &storage.NetworkPolicy{
						Id:   id,
						Name: fmt.Sprintf("networkpolicy-%s", id[:8]),
					},
				}
			},
		},
		{
			name:       "Secret",
			percentage: 10,
			generator: func(id string) interface{} {
				return &central.SensorEvent_Secret{
					Secret: &storage.Secret{
						Id:   id,
						Name: fmt.Sprintf("secret-%s", id[:8]),
					},
				}
			},
		},
		{
			name:       "ServiceAccount",
			percentage: 10,
			generator: func(id string) interface{} {
				return &central.SensorEvent_ServiceAccount{
					ServiceAccount: &storage.ServiceAccount{
						Id:   id,
						Name: fmt.Sprintf("serviceaccount-%s", id[:8]),
					},
				}
			},
		},
		{
			name:       "Role",
			percentage: 10,
			generator: func(id string) interface{} {
				return &central.SensorEvent_Role{
					Role: &storage.K8SRole{
						Id:   id,
						Name: fmt.Sprintf("role-%s", id[:8]),
					},
				}
			},
		},
		{
			name:       "Binding",
			percentage: 10,
			generator: func(id string) interface{} {
				return &central.SensorEvent_Binding{
					Binding: &storage.K8SRoleBinding{
						Id:   id,
						Name: fmt.Sprintf("binding-%s", id[:8]),
					},
				}
			},
		},
	}
}

// generateHashKeyForType generates a hash key based on the sensor event type
func generateHashKeyForType(typeDistribution sensorEventTypeDistribution, id string) string {
	// Format: <TYPE>:<UUID> as per the deduper key format
	return fmt.Sprintf("%s:%s", typeDistribution.name, id)
}

// generateHashesForCluster generates hashes for a single cluster with the specified distribution
func generateHashesForCluster(clusterID string, hashesPerCluster int) map[string]uint64 {
	hashes := make(map[string]uint64)
	types := generateSensorEventTypes()

	// Calculate how many hashes per type based on percentages
	hashesPerType := make(map[int]int)
	for i, t := range types {
		hashesPerType[i] = (hashesPerCluster * t.percentage) / 100
	}

	// Distribute any remainder to the first type
	totalAllocated := 0
	for _, count := range hashesPerType {
		totalAllocated += count
	}
	if totalAllocated < hashesPerCluster {
		hashesPerType[0] += hashesPerCluster - totalAllocated
	}

	// Generate hashes
	hashCounter := uint64(1000)
	for typeIdx, count := range hashesPerType {
		for i := 0; i < count; i++ {
			id := uuid.NewV4().String()
			key := generateHashKeyForType(types[typeIdx], id)
			hashes[key] = hashCounter
			hashCounter++
		}
	}

	return hashes
}

func BenchmarkHashDatastoreOperations(b *testing.B) {
	// Enable the feature flag for hash storage
	b.Setenv(features.StoreEventHashes.EnvVar(), "true")
	if !features.StoreEventHashes.Enabled() {
		b.Skip("Skip hash datastore benchmark because feature flag is off")
	}

	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)
	datastore := GetTestPostgresDataStore(b, testDB.DB)

	const (
		numClusters      = 100
		hashesPerCluster = 80000
	)

	// Prepare data: generate hashes for all clusters
	clusterData := make(map[string]map[string]uint64)
	for i := 0; i < numClusters; i++ {
		clusterID := uuid.NewV4().String()
		hashes := generateHashesForCluster(clusterID, hashesPerCluster)
		clusterData[clusterID] = hashes
	}

	b.ResetTimer()

	// Benchmark: UpsertHash operations
	b.Run("UpsertHashes", func(b *testing.B) {
		b.ReportAllocs()
		// Reset timer after data generation in parent benchmark
		for i := 0; i < b.N; i++ {
			for clusterID, hashes := range clusterData {
				hash := &storage.Hash{
					ClusterId: clusterID,
					Hashes:    hashes,
				}
				err := datastore.UpsertHash(ctx, hash)
				require.NoError(b, err)
			}
		}
	})

	b.Run("GetHashes", func(b *testing.B) {
		b.ReportAllocs()
		// First, upsert all hashes
		for clusterID, hashes := range clusterData {
			hash := &storage.Hash{
				ClusterId: clusterID,
				Hashes:    hashes,
			}
			err := datastore.UpsertHash(ctx, hash)
			require.NoError(b, err)
		}
		b.ResetTimer()

		// Now benchmark GetHashes
		for i := 0; i < b.N; i++ {
			for clusterID := range clusterData {
				_, exists, err := datastore.GetHashes(ctx, clusterID)
				require.NoError(b, err)
				require.True(b, exists)
			}
		}
	})
}

// BenchmarkHashDistributionVerification verifies that the hash distribution matches the requirements
func TestHashDistributionVerification(t *testing.T) {
	const hashesPerCluster = 80000
	hashes := generateHashesForCluster(uuid.NewV4().String(), hashesPerCluster)

	types := generateSensorEventTypes()
	typeCount := make(map[string]int)

	for key := range hashes {
		// Parse the key to get the type (format: TYPE:UUID)
		for _, typeInfo := range types {
			if fmt.Sprintf("%s:", typeInfo.name) == key[:len(typeInfo.name)+1] {
				typeCount[typeInfo.name]++
				break
			}
		}
	}

	// Verify distribution (allowing some tolerance due to rounding)
	tolerance := 0.02 // 2% tolerance
	for _, typeInfo := range types {
		count := typeCount[typeInfo.name]
		expectedCount := (hashesPerCluster * typeInfo.percentage) / 100
		actualPercentage := float64(count) / float64(hashesPerCluster) * 100
		expectedPercentage := float64(typeInfo.percentage)

		diff := actualPercentage - expectedPercentage
		if diff < 0 {
			diff = -diff
		}

		t.Logf("Type: %s, Expected: %d (%.1f%%), Actual: %d (%.1f%%), Diff: %.2f%%",
			typeInfo.name, expectedCount, expectedPercentage, count, actualPercentage, diff)

		if diff > tolerance*100 {
			t.Errorf("Type %s distribution is off: expected %.1f%%, got %.1f%%",
				typeInfo.name, expectedPercentage, actualPercentage)
		}
	}

	// Verify total count
	totalCount := 0
	for _, count := range typeCount {
		totalCount += count
	}
	if totalCount != hashesPerCluster {
		t.Errorf("Total hash count mismatch: expected %d, got %d", hashesPerCluster, totalCount)
	}
}
