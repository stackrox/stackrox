package idmap

import (
	"context"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stackrox/rox/central/namespace/store/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

const (
	numParallelGoroutines = 20
)

var (
	namespaceCounts = []int{10, 100, 1000, 10000}
)

func BenchmarkSharedIDMapStorage_LookupsSingleThread(b *testing.B) {
	for _, numNamespaces := range namespaceCounts {
		namespaces := make([]*storage.NamespaceMetadata, 0, numNamespaces)
		for i := 0; i < numNamespaces; i++ {
			namespaces = append(namespaces, &storage.NamespaceMetadata{
				Id:          uuid.NewV4().String(),
				Name:        uuid.NewV4().String(),
				ClusterId:   uuid.NewV4().String(),
				ClusterName: uuid.NewV4().String(),
			})
		}

		b.Run(strconv.Itoa(numNamespaces), func(b *testing.B) {
			s := newSharedIDMapStorage()
			s.OnNamespaceAdd(namespaces...)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ns := namespaces[rand.Int()%numNamespaces]
				nsInfo := s.Get().ByNamespaceID(ns.GetId())
				require.NotNil(b, nsInfo)
				require.Equal(b, ns.GetClusterName(), nsInfo.ClusterName)
			}
		})
	}
}

func BenchmarkCrudStorage_LookupsSingleThread(b *testing.B) {
	ctx := context.Background()
	for _, numNamespaces := range namespaceCounts {
		namespaces := make([]*storage.NamespaceMetadata, 0, numNamespaces)
		for i := 0; i < numNamespaces; i++ {
			namespaces = append(namespaces, &storage.NamespaceMetadata{
				Id:          uuid.NewV4().String(),
				Name:        uuid.NewV4().String(),
				ClusterId:   uuid.NewV4().String(),
				ClusterName: uuid.NewV4().String(),
			})
		}

		b.Run(strconv.Itoa(numNamespaces), func(b *testing.B) {
			rocksDB := rocksdbtest.RocksDBForT(b)
			defer func() {
				b.StopTimer()
				rocksdbtest.TearDownRocksDB(rocksDB)
			}()

			s := rocksdb.New(rocksDB)
			require.NoError(b, s.UpsertMany(ctx, namespaces))

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ns := namespaces[rand.Int()%numNamespaces]
				nsInfo, _, _ := s.Get(ctx, ns.GetId())
				require.NotNil(b, nsInfo)
				require.Equal(b, ns.GetClusterName(), nsInfo.ClusterName)
			}
		})
	}
}

func BenchmarkSharedIDMapStorage_LookupsMultiThread(b *testing.B) {
	for _, numNamespaces := range namespaceCounts {
		namespaces := make([]*storage.NamespaceMetadata, 0, numNamespaces)
		for i := 0; i < numNamespaces; i++ {
			namespaces = append(namespaces, &storage.NamespaceMetadata{
				Id:          uuid.NewV4().String(),
				Name:        uuid.NewV4().String(),
				ClusterId:   uuid.NewV4().String(),
				ClusterName: uuid.NewV4().String(),
			})
		}

		b.Run(strconv.Itoa(numNamespaces), func(b *testing.B) {
			s := newSharedIDMapStorage()
			s.OnNamespaceAdd(namespaces...)

			b.ResetTimer()
			var wg sync.WaitGroup
			for j := 0; j < numParallelGoroutines; j++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					for i := 0; i < b.N; i++ {
						ns := namespaces[rand.Int()%numNamespaces]
						nsInfo := s.Get().ByNamespaceID(ns.GetId())
						require.NotNil(b, nsInfo)
						require.Equal(b, ns.GetClusterName(), nsInfo.ClusterName)
					}
				}()
			}
			wg.Wait()
		})
	}
}

func BenchmarkCrudStorage_LookupsMultiThread(b *testing.B) {
	ctx := context.Background()
	for _, numNamespaces := range namespaceCounts {
		namespaces := make([]*storage.NamespaceMetadata, 0, numNamespaces)
		for i := 0; i < numNamespaces; i++ {
			namespaces = append(namespaces, &storage.NamespaceMetadata{
				Id:          uuid.NewV4().String(),
				Name:        uuid.NewV4().String(),
				ClusterId:   uuid.NewV4().String(),
				ClusterName: uuid.NewV4().String(),
			})
		}

		b.Run(strconv.Itoa(numNamespaces), func(b *testing.B) {
			rocksDB := rocksdbtest.RocksDBForT(b)
			defer func() {
				b.StopTimer()
				rocksdbtest.TearDownRocksDB(rocksDB)
			}()

			s := rocksdb.New(rocksDB)
			require.NoError(b, s.UpsertMany(ctx, namespaces))

			b.ResetTimer()
			var wg sync.WaitGroup
			for j := 0; j < numParallelGoroutines; j++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					for i := 0; i < b.N; i++ {
						ns := namespaces[rand.Int()%numNamespaces]
						nsInfo, _, _ := s.Get(ctx, ns.GetId())
						require.NotNil(b, nsInfo)
						require.Equal(b, ns.GetClusterName(), nsInfo.ClusterName)
					}
				}()
			}
			wg.Wait()
		})
	}
}
