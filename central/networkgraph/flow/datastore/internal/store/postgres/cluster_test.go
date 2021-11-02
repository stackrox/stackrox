package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
)

func TestT(t *testing.T) {
	source := "host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000 pool_min_conns=90 pool_max_conns=90"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(source)
	if err != nil {
		panic(err)
	}
	pool, err := pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		panic(err)
	}

	fs := NewClusterStore(pool).GetFlowStore("clusterid2")

	t1 := time.Now().Add(-5 * time.Minute)
	t2 := time.Now()
	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode2"},
				DstPort:    1,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(t1),
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"},
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(t2),
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "yetAnotherNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "yetAnotherNode2"},
				DstPort:    3,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
		},
	}
	updateTS := timestamp.Now() - 1000000
	if err := fs.UpsertFlows(flows, updateTS); err != nil {
		panic(err)
	}
}
