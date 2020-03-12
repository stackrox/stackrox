package badger

import (
	"testing"

	"github.com/stackrox/rox/central/pod/store"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/require"
)

func getPodStore(b *testing.B) store.Store {
	db, _, err := badgerhelper.NewTemp(b.Name() + ".db")
	if err != nil {
		b.Fatal(err)
	}

	return New(db)
}

func BenchmarkAddPod(b *testing.B) {
	store := getPodStore(b)
	pod := fixtures.GetPod()
	for i := 0; i < b.N; i++ {
		require.NoError(b, store.UpsertPod(pod))
	}
}

func BenchmarkGetPod(b *testing.B) {
	store := getPodStore(b)
	pod := fixtures.GetPod()
	require.NoError(b, store.UpsertPod(pod))
	for i := 0; i < b.N; i++ {
		_, exists, err := store.GetPod(pod.GetId())
		require.True(b, exists)
		require.NoError(b, err)
	}
}
