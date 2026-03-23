package gc

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

// testStore is a minimal implementation of the Store interface for testing.
type testStore struct {
	calls   int
	results []int64
}

// DeleteOrphanedCVEsBatch is the only method we need to implement for the test.
func (s *testStore) DeleteOrphanedCVEsBatch(ctx context.Context, batchSize int) (int64, error) {
	if s.calls >= len(s.results) {
		return 0, nil
	}
	result := s.results[s.calls]
	s.calls++
	return result, nil
}

// Unused Store methods (panics to ensure they're not called).
func (s *testStore) UpsertCVE(ctx context.Context, cveRow *store.CVERow) (string, error) {
	panic("not implemented")
}

func (s *testStore) UpsertEdge(ctx context.Context, edge *store.EdgeRow) error {
	panic("not implemented")
}

func (s *testStore) DeleteStaleEdges(ctx context.Context, componentID string, keepCVEIDs []string) error {
	panic("not implemented")
}

func (s *testStore) GetCVEsForImage(ctx context.Context, imageID string) ([]*store.CVERow, error) {
	panic("not implemented")
}

func (s *testStore) GetAllReferencedCVEs(ctx context.Context) ([]*store.CVERow, error) {
	panic("not implemented")
}

func (s *testStore) Count(_ context.Context, _ *v1.Query) (int, error) {
	panic("not implemented")
}

func (s *testStore) Exists(_ context.Context, _ string) (bool, error) {
	panic("not implemented")
}

func (s *testStore) GetIDs(_ context.Context) ([]string, error) {
	panic("not implemented")
}

func TestRunOnce_MultiplePartialBatches(t *testing.T) {
	// Test with multiple batches where the second batch is partial.
	ts := &testStore{
		results: []int64{1000, 500}, // First call returns 1000, second returns 500.
	}

	mgr := New(ts)
	ctx := context.Background()

	total, err := mgr.RunOnce(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1500), total, "Expected total of 1500 deleted CVEs")
	assert.Equal(t, 2, ts.calls, "Expected DeleteOrphanedCVEsBatch to be called twice")
}

func TestRunOnce_EmptyResult(t *testing.T) {
	// Test when no orphaned CVEs exist.
	ts := &testStore{
		results: []int64{0}, // No orphans to delete.
	}

	mgr := New(ts)
	ctx := context.Background()

	total, err := mgr.RunOnce(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total, "Expected total of 0 deleted CVEs")
	assert.Equal(t, 1, ts.calls, "Expected DeleteOrphanedCVEsBatch to be called once")
}

func TestRunOnce_MaxBatchesLimit(t *testing.T) {
	// Test that we respect the max batches limit (gcMaxBatches = 100).
	// Create a store that would return 1000 every time (simulating many orphans).
	results := make([]int64, 200) // More than gcMaxBatches.
	for i := range results {
		results[i] = 1000
	}
	ts := &testStore{
		results: results,
	}

	mgr := New(ts)
	ctx := context.Background()

	total, err := mgr.RunOnce(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(gcMaxBatches*1000), total, "Expected total to be limited by gcMaxBatches")
	assert.Equal(t, gcMaxBatches, ts.calls, "Expected DeleteOrphanedCVEsBatch to be called gcMaxBatches times")
}
