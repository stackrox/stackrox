package gc

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

// testDataStore is a minimal implementation of the DataStore interface for testing.
// Only DeleteOrphanedCVEsBatch has real behavior; all other methods panic.
type testDataStore struct {
	calls   int
	results []int64
}

// DeleteOrphanedCVEsBatch is the only method exercised by the GC manager.
func (s *testDataStore) DeleteOrphanedCVEsBatch(ctx context.Context, batchSize int) (int64, error) {
	if s.calls >= len(s.results) {
		return 0, nil
	}
	result := s.results[s.calls]
	s.calls++
	return result, nil
}

// All other methods are not used by GC, panic to catch accidental calls.

func (s *testDataStore) Search(_ context.Context, _ *v1.Query) ([]search.Result, error) {
	panic("not implemented")
}

func (s *testDataStore) Count(_ context.Context, _ *v1.Query) (int, error) {
	panic("not implemented")
}

func (s *testDataStore) SearchImageCVEs(_ context.Context, _ *v1.Query) ([]*v1.SearchResult, error) {
	panic("not implemented")
}

func (s *testDataStore) SearchRawImageCVEs(_ context.Context, _ *v1.Query) ([]*storage.ImageCVEV2, error) {
	panic("not implemented")
}

func (s *testDataStore) GetBatch(_ context.Context, _ []string) ([]*storage.ImageCVEV2, error) {
	panic("not implemented")
}

func (s *testDataStore) Exists(_ context.Context, _ string) (bool, error) {
	panic("not implemented")
}

func (s *testDataStore) Get(_ context.Context, _ string) (*storage.NormalizedCVE, bool, error) {
	panic("not implemented")
}

func (s *testDataStore) Upsert(_ context.Context, _ *storage.NormalizedCVE) error {
	panic("not implemented")
}

func (s *testDataStore) UpsertEdge(_ context.Context, _ *storage.NormalizedComponentCVEEdge) error {
	panic("not implemented")
}

func (s *testDataStore) DeleteStaleEdges(_ context.Context, _ string, _ []string) error {
	panic("not implemented")
}

func (s *testDataStore) GetCVEsForImage(_ context.Context, _ string) ([]*storage.NormalizedCVE, error) {
	panic("not implemented")
}

func (s *testDataStore) GetAllReferencedCVEs(_ context.Context) ([]*storage.NormalizedCVE, error) {
	panic("not implemented")
}

func TestRunOnce_MultiplePartialBatches(t *testing.T) {
	// Test with multiple batches where the second batch is partial.
	ts := &testDataStore{
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
	ts := &testDataStore{
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
	ts := &testDataStore{
		results: results,
	}

	mgr := New(ts)
	ctx := context.Background()

	total, err := mgr.RunOnce(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(gcMaxBatches*1000), total, "Expected total to be limited by gcMaxBatches")
	assert.Equal(t, gcMaxBatches, ts.calls, "Expected DeleteOrphanedCVEsBatch to be called gcMaxBatches times")
}
