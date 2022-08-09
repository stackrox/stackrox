package dackbox

import (
	"context"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

const maxGRPCSize = 4194304

func getDeploymentStore(b *testing.B) *StoreImpl {
	db, err := rocksdb.NewTemp("reference")
	if err != nil {
		b.Fatal(err)
	}
	dacky, err := dackbox.NewRocksDBDackBox(db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	if err != nil {
		b.Fatal(err)
	}
	s := New(dacky, concurrency.NewKeyFence())

	return s
}

func BenchmarkAddDeployment(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())

	store := getDeploymentStore(b)
	deployment := fixtures.GetDeployment()
	for i := 0; i < b.N; i++ {
		require.NoError(b, store.Upsert(ctx, deployment))
	}
}

func BenchmarkGetDeployment(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())

	store := getDeploymentStore(b)
	deployment := fixtures.GetDeployment()
	require.NoError(b, store.Upsert(ctx, deployment))
	for i := 0; i < b.N; i++ {
		_, exists, err := store.Get(ctx, deployment.GetId())
		require.True(b, exists)
		require.NoError(b, err)
	}
}

func BenchmarkListDeployment(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())

	store := getDeploymentStore(b)
	deployment := fixtures.GetDeployment()
	require.NoError(b, store.Upsert(ctx, deployment))
	for i := 0; i < b.N; i++ {
		_, exists, err := store.GetListDeployment(ctx, deployment.GetId())
		require.True(b, exists)
		require.NoError(b, err)
	}
}

// This really isn't a benchmark, but just prints out how many ListDeployments can be returned in an API call
func BenchmarkListDeployments(b *testing.B) {
	listDeployment := &storage.ListDeployment{
		Id:        uuid.NewDummy().String(),
		Name:      "quizzical_cat",
		Cluster:   "Production k8s",
		Namespace: "stackrox",
		Created:   types.TimestampNow(),
		Priority:  10,
	}

	bytes, _ := proto.Marshal(listDeployment)
	fmt.Printf("Max ListDeployments that can be returned: %d\n", maxGRPCSize/len(bytes))
}
