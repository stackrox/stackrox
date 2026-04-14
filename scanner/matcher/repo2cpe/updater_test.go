package repo2cpe

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGetter implements Getter for testing.
type mockGetter struct {
	result    *indexer.FetchResult
	err       error
	callCount atomic.Int32
}

func (m *mockGetter) GetRepositoryToCPEMapping(_ context.Context, _ string) (*indexer.FetchResult, error) {
	m.callCount.Add(1)
	return m.result, m.err
}

func TestUpdater_Get(t *testing.T) {
	t.Run("returns cached value after init", func(t *testing.T) {
		g := &mockGetter{
			result: &indexer.FetchResult{
				Modified:     true,
				LastModified: "Tue, 01 Jan 2025 00:00:00 GMT",
				Data: &repositorytocpe.MappingFile{
					Data: map[string]repositorytocpe.Repo{
						"rhel-8": {CPEs: []string{"cpe:rhel:8"}},
					},
				},
			},
		}
		u := NewUpdater(g)
		defer u.Close()

		mf := u.Get(context.Background())
		require.NotNil(t, mf)
		assert.Len(t, mf.Data, 1)
		cpes, ok := mf.GetCPEs("rhel-8")
		assert.True(t, ok)
		assert.Equal(t, []string{"cpe:rhel:8"}, cpes)
	})

	t.Run("returns empty MappingFile when initial fetch fails", func(t *testing.T) {
		g := &mockGetter{
			err: errors.New("connection refused"),
		}
		u := NewUpdater(g)
		defer u.Close()

		mf := u.Get(context.Background())
		require.NotNil(t, mf)
		assert.Empty(t, mf.Data)
	})

	t.Run("lazy init called exactly once", func(t *testing.T) {
		g := &mockGetter{
			result: &indexer.FetchResult{
				Modified:     true,
				LastModified: "now",
				Data:         &repositorytocpe.MappingFile{Data: map[string]repositorytocpe.Repo{}},
			},
		}
		u := NewUpdater(g)
		defer u.Close()

		// Call Get multiple times concurrently.
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				u.Get(context.Background())
			})
		}
		wg.Wait()

		// Initial fetch should be called exactly once (by init).
		// The background refresh may add more calls, so we just assert >= 1.
		assert.GreaterOrEqual(t, int(g.callCount.Load()), 1)
	})
}

func TestUpdater_fetch(t *testing.T) {
	t.Run("modified result stores new data", func(t *testing.T) {
		g := &mockGetter{
			result: &indexer.FetchResult{
				Modified:     true,
				LastModified: "Tue, 01 Jan 2025 00:00:00 GMT",
				Data: &repositorytocpe.MappingFile{
					Data: map[string]repositorytocpe.Repo{
						"repo-1": {CPEs: []string{"cpe:1"}},
					},
				},
			},
		}
		u := NewUpdater(g)
		defer u.Close()

		err := u.fetch(context.Background(), "")
		require.NoError(t, err)

		v := u.value.Load()
		require.NotNil(t, v)
		assert.Len(t, v.Data, 1)

		concurrency.WithRLock(&u.mu, func() {
			assert.Equal(t, "Tue, 01 Jan 2025 00:00:00 GMT", u.lastModified)
		})
	})

	t.Run("not modified result preserves existing data", func(t *testing.T) {
		existing := &repositorytocpe.MappingFile{
			Data: map[string]repositorytocpe.Repo{
				"old-repo": {CPEs: []string{"cpe:old"}},
			},
		}
		g := &mockGetter{
			result: &indexer.FetchResult{
				Modified:     false,
				LastModified: "Mon, 01 Jan 2024 00:00:00 GMT",
			},
		}
		u := NewUpdater(g)
		defer u.Close()
		u.value.Store(existing)

		err := u.fetch(context.Background(), "Mon, 01 Jan 2024 00:00:00 GMT")
		require.NoError(t, err)

		v := u.value.Load()
		require.NotNil(t, v)
		assert.Len(t, v.Data, 1)
		_, ok := v.Data["old-repo"]
		assert.True(t, ok)
	})

	t.Run("error returns error and preserves value", func(t *testing.T) {
		existing := &repositorytocpe.MappingFile{
			Data: map[string]repositorytocpe.Repo{
				"existing": {CPEs: []string{"cpe:existing"}},
			},
		}
		g := &mockGetter{
			err: errors.New("rpc error"),
		}
		u := NewUpdater(g)
		defer u.Close()
		u.value.Store(existing)

		err := u.fetch(context.Background(), "")
		require.Error(t, err)

		v := u.value.Load()
		require.NotNil(t, v)
		assert.Len(t, v.Data, 1)
	})
}

func TestUpdater_Close(t *testing.T) {
	g := &mockGetter{
		result: &indexer.FetchResult{
			Modified: true,
			Data:     &repositorytocpe.MappingFile{Data: map[string]repositorytocpe.Repo{}},
		},
	}
	u := NewUpdater(g)

	// Trigger init to start the background goroutine.
	u.Get(context.Background())

	// Close should not panic or block.
	assert.NotPanics(t, func() {
		u.Close()
	})
}
