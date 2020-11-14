package datastore

import (
	"context"
	"net"
	"testing"

	"github.com/stackrox/rox/central/networkgraph/config/datastore"
	store "github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
)

func BenchmarkNetEntityCreation(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
		))

	db, err := rocksdb.NewTemp(b.Name())
	require.NoError(b, err)
	defer rocksdbtest.TearDownRocksDB(db)

	store, err := store.New(db)
	require.NoError(b, err)

	ds := NewEntityDataStore(store, datastore.Singleton(), networktree.Singleton(), connection.ManagerSingleton())

	ip := net.ParseIP("1.1.1.1")
	entities := make([]*storage.NetworkEntity, 10000)
	for i := 0; i < 10000; i++ {
		cidr := ip.String() + "/32"
		id, _ := externalsrcs.NewClusterScopedID("c1", cidr)
		e := &storage.NetworkEntity{
			Info: &storage.NetworkEntityInfo{
				Id:   id.String(),
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name: cidr,
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
							Cidr: cidr,
						},
					},
				},
			},
			Scope: &storage.NetworkEntity_Scope{
				ClusterId: "c1",
			},
		}
		entities[i] = e
		ip = nextIP(ip)
	}

	b.Run("upsertNetworkEntities", func(b *testing.B) {
		for _, e := range entities {
			require.NoError(b, ds.CreateExternalNetworkEntity(ctx, e, true))
		}
	})

}

func nextIP(ip net.IP) net.IP {
	i := ip.To4()
	v := uint(i[0])<<24 + uint(i[1])<<16 + uint(i[2])<<8 + uint(i[3])
	v++
	v3 := byte(v & 0xFF)
	v2 := byte((v >> 8) & 0xFF)
	v1 := byte((v >> 16) & 0xFF)
	v0 := byte((v >> 24) & 0xFF)
	return net.IPv4(v0, v1, v2, v3)
}
