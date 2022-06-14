package mapcache

import (
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
)

func BenchmarkWalk(b *testing.B) {
	cluster := &storage.Cluster{
		Type:               storage.ClusterType_KUBERNETES_CLUSTER,
		MainImage:          "stackrox/main:latest",
		CollectorImage:     "stackrox/collector",
		CentralApiEndpoint: "central.stackrox:443",
		Status: &storage.ClusterStatus{
			SensorVersion: "sd",
		},
		DynamicConfig: &storage.DynamicClusterConfig{
			AdmissionControllerConfig: &storage.AdmissionControllerConfig{
				Enabled:          false,
				TimeoutSeconds:   3,
				ScanInline:       false,
				DisableBypass:    false,
				EnforceOnUpdates: false,
			},
		},
		HealthStatus: &storage.ClusterHealthStatus{
			SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
			CollectorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
			OverallHealthStatus:   storage.ClusterHealthStatus_HEALTHY,
		},
	}
	db, err := rocksdb.NewTemp(b.Name())
	require.NoError(b, err)
	defer rocksdbtest.TearDownRocksDB(db)

	keyFunc := func(msg proto.Message) []byte {
		return []byte(msg.(*storage.Cluster).GetId())
	}
	alloc := func() proto.Message {
		return &storage.Cluster{}
	}

	baseCrud := generic.NewCRUD(db, []byte("cluster"), keyFunc, alloc, false)
	crud, err := NewMapCache(baseCrud, keyFunc)
	require.NoError(b, err)

	for i := 0; i < 10000; i++ {
		cluster.Id = strconv.Itoa(i)
		cluster.Name = strconv.Itoa(i)
		err := crud.Upsert(cluster)
		require.NoError(b, err)
	}

	b.Run("walk", func(b *testing.B) {
		err := crud.Walk(func(msg proto.Message) error {
			return nil
		})
		require.NoError(b, err)
	})
}
