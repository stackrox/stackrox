package idmap

import (
	"math/rand"
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readAndManipulate(t *testing.T, wg *sync.WaitGroup, s *sharedIDMapStorage, numOps, readToWriteRatio int) {
	defer wg.Done()

	var origPointers, snapshots []*IDMap

	for i := 0; i < numOps; i++ {
		if rand.Int()%(readToWriteRatio+1) == 0 { // perform a write
			s.OnNamespaceAdd(&storage.NamespaceMetadata{
				Id:          uuid.NewV4().String(),
				Name:        uuid.NewV4().String(),
				ClusterId:   uuid.NewV4().String(),
				ClusterName: uuid.NewV4().String(),
			})
			continue
		}

		ptr := s.Get()
		if len(origPointers) == 0 || origPointers[len(origPointers)-1] != ptr {
			origPointers = append(origPointers, ptr)
			snapshots = append(snapshots, ptr.Clone())
		}
	}

	require.Equal(t, len(origPointers), len(snapshots))

	// Verify that the sequence of snapshots and original pointers are point-wisely equal.
	for i, origPtr := range origPointers {
		snapshot := snapshots[i]
		assert.NotSame(t, origPtr, snapshot)
		assert.NotSame(t, origPtr.byNamespaceID, snapshot.byNamespaceID)
		assert.Equal(t, origPtr.byNamespaceID, snapshot.byNamespaceID)
	}
}

// Launch a number of goroutines that each randomly performs a read or a write in each operation. Every time
// a new pointer is returned, a snapshot is taken. At the end, we ensure that the stored pointers and their
// snapshots match.
// Tested with go test -race -p 4 -count 10
func TestSharedStorage_ConcurrencySafe(t *testing.T) {
	t.Parallel()

	const (
		numGoroutines      = 100
		numOpsPerGoroutine = 10000
		readToWriteRatio   = 1000
	)

	var wg sync.WaitGroup
	s := newSharedIDMapStorage()
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			readAndManipulate(t, &wg, s, numOpsPerGoroutine, readToWriteRatio)
		}()
	}
	wg.Wait()
}

func TestSharedStorage(t *testing.T) {
	t.Parallel()

	s := newSharedIDMapStorage()
	s.OnNamespaceAdd(&storage.NamespaceMetadata{
		Id:          "id1",
		Name:        "name1",
		ClusterId:   "clusterid1",
		ClusterName: "clustername1",
	})

	m := s.Get()
	require.NotNil(t, m)

	// Verify that we see what was added.
	assert.Len(t, m.byNamespaceID, 1)
	info := m.byNamespaceID["id1"]
	require.NotNil(t, info)
	assert.Equal(t, "id1", info.ID)
	assert.Equal(t, "name1", info.Name)
	assert.Equal(t, "clusterid1", info.ClusterID)
	assert.Equal(t, "clustername1", info.ClusterName)

	m2 := s.Get()

	// Verify that a subsequent get sees the same pointer
	assert.Same(t, m, m2)

	// Perform an update.
	var seenDuringUpdate *IDMap
	s.Update(func(m *IDMap) bool {
		seenDuringUpdate = m
		m.byNamespaceID["id2"] = &NamespaceInfo{
			ID:          "id2",
			Name:        "name2",
			ClusterID:   "clusterid1",
			ClusterName: "clustername1",
		}
		return true
	})

	// Verify updater ran
	require.NotNil(t, seenDuringUpdate)

	var seenDuringNextUpdate *IDMap
	s.Update(func(m *IDMap) bool {
		seenDuringNextUpdate = m
		m.byNamespaceID["id3"] = &NamespaceInfo{
			ID:          "id3",
			Name:        "name3",
			ClusterID:   "clusterid1",
			ClusterName: "clustername1",
		}
		return true
	})

	require.NotNil(t, seenDuringNextUpdate)

	// Verify updaters operated on the same object.
	assert.Same(t, seenDuringUpdate, seenDuringNextUpdate)

	// After calling Get() after an update, we should get the element that was operated on by the last update.
	seenPostUpdate := s.Get()
	assert.Same(t, seenDuringUpdate, seenPostUpdate)

	// When doing another update, we should now operate on a different element. However, since the update has no
	// effect, this should not change what we see in `Get`.
	s.Update(func(m *IDMap) bool {
		seenDuringUpdate = m
		return false
	})
	require.NotNil(t, seenDuringUpdate)

	assert.NotSame(t, seenPostUpdate, seenDuringUpdate)

	seenPostNoOpUpdate := s.Get()
	assert.Same(t, seenPostUpdate, seenPostNoOpUpdate)
}
