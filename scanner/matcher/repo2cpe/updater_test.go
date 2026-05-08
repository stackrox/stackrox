package repo2cpe

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stackrox/rox/scanner/matcher/repo2cpe/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUpdater_Get(t *testing.T) {
	t.Run("returns cached value after init", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)
		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(&indexer.FetchResult{
				Modified:     true,
				LastModified: "Tue, 01 Jan 2025 00:00:00 GMT",
				Data: &repositorytocpe.MappingFile{
					Data: map[string]repositorytocpe.Repo{
						"rhel-8": {CPEs: []string{"cpe:rhel:8"}},
					},
				},
			}, nil).
			AnyTimes()

		u := NewUpdater(g)
		defer u.Close()

		mf, err := u.Get(context.Background())
		require.NoError(t, err)
		require.NotNil(t, mf)
		assert.Len(t, mf.Data, 1)
		cpes, ok := mf.GetCPEs("rhel-8")
		assert.True(t, ok)
		assert.Equal(t, []string{"cpe:rhel:8"}, cpes)
	})

	t.Run("returns error when all fetches fail", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)
		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("connection refused")).
			AnyTimes()

		u := NewUpdater(g)
		defer u.Close()

		mf, err := u.Get(context.Background())
		require.Error(t, err)
		assert.ErrorIs(t, err, errNoSuccessfulFetch)
		assert.Nil(t, mf)
	})

	t.Run("lazy init called exactly once", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)
		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(&indexer.FetchResult{
				Modified:     true,
				LastModified: "now",
				Data:         &repositorytocpe.MappingFile{Data: map[string]repositorytocpe.Repo{}},
			}, nil).
			MinTimes(1)

		u := NewUpdater(g)
		defer u.Close()

		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				_, err := u.Get(context.Background())
				assert.NoError(t, err)
			})
		}
		wg.Wait()
	})
}

func TestUpdater_fetch(t *testing.T) {
	t.Run("modified result stores new data", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)
		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(&indexer.FetchResult{
				Modified:     true,
				LastModified: "Tue, 01 Jan 2025 00:00:00 GMT",
				Data: &repositorytocpe.MappingFile{
					Data: map[string]repositorytocpe.Repo{
						"repo-1": {CPEs: []string{"cpe:1"}},
					},
				},
			}, nil)

		u := NewUpdater(g)
		defer u.Close()

		err := u.fetch(context.Background(), "")
		require.NoError(t, err)
		assert.False(t, u.lastFailed.Load())

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
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)
		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(&indexer.FetchResult{
				Modified:     false,
				LastModified: "Mon, 01 Jan 2024 00:00:00 GMT",
			}, nil)

		u := NewUpdater(g)
		defer u.Close()
		u.value.Store(existing)

		err := u.fetch(context.Background(), "Mon, 01 Jan 2024 00:00:00 GMT")
		require.NoError(t, err)
		assert.False(t, u.lastFailed.Load())

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
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)
		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("rpc error"))

		u := NewUpdater(g)
		defer u.Close()
		u.value.Store(existing)

		err := u.fetch(context.Background(), "")
		require.Error(t, err)
		assert.True(t, u.lastFailed.Load())

		v := u.value.Load()
		require.NotNil(t, v)
		assert.Len(t, v.Data, 1)
	})

	t.Run("modified result with nil data returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)
		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(&indexer.FetchResult{
				Modified:     true,
				LastModified: "now",
				Data:         nil,
			}, nil)

		u := NewUpdater(g)
		defer u.Close()

		err := u.fetch(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil data")
		assert.Nil(t, u.value.Load())
	})

	t.Run("lastFailed cleared after successful fetch following failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)

		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("transient error"))

		u := NewUpdater(g)
		defer u.Close()

		_ = u.fetch(context.Background(), "")
		assert.True(t, u.lastFailed.Load())

		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(&indexer.FetchResult{
				Modified:     true,
				LastModified: "now",
				Data:         &repositorytocpe.MappingFile{Data: map[string]repositorytocpe.Repo{}},
			}, nil)

		err := u.fetch(context.Background(), "")
		require.NoError(t, err)
		assert.False(t, u.lastFailed.Load())
	})
}

func TestUpdater_refreshLoop(t *testing.T) {
	t.Run("uses shorter interval after failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)

		callCount := 0
		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string) (*indexer.FetchResult, error) {
				callCount++
				if callCount <= 2 {
					return nil, errors.New("transient error")
				}
				return &indexer.FetchResult{
					Modified:     true,
					LastModified: "now",
					Data:         &repositorytocpe.MappingFile{Data: map[string]repositorytocpe.Repo{}},
				}, nil
			}).
			MinTimes(3)

		u := NewUpdater(g)
		u.failedInterval = 50 * time.Millisecond
		u.lastFailed.Store(true)

		ctx := context.Background()
		go u.refreshLoop(ctx)

		assert.Eventually(t, func() bool {
			return u.value.Load() != nil
		}, 2*time.Second, 10*time.Millisecond)

		u.Close()
	})
}

func TestUpdater_Close(t *testing.T) {
	t.Run("does not panic", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)
		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(&indexer.FetchResult{
				Modified: true,
				Data:     &repositorytocpe.MappingFile{Data: map[string]repositorytocpe.Repo{}},
			}, nil).
			AnyTimes()

		u := NewUpdater(g)

		_, err := u.Get(context.Background())
		require.NoError(t, err)

		assert.NotPanics(t, func() {
			u.Close()
		})
	})

	t.Run("multiple calls do not panic", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)
		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(&indexer.FetchResult{
				Modified: true,
				Data:     &repositorytocpe.MappingFile{Data: map[string]repositorytocpe.Repo{}},
			}, nil).
			AnyTimes()

		u := NewUpdater(g)
		_, err := u.Get(context.Background())
		require.NoError(t, err)

		assert.NotPanics(t, func() {
			u.Close()
			u.Close()
			u.Close()
		})
	})

	t.Run("Get after Close returns cached value", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)
		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(&indexer.FetchResult{
				Modified:     true,
				LastModified: "now",
				Data: &repositorytocpe.MappingFile{
					Data: map[string]repositorytocpe.Repo{
						"repo": {CPEs: []string{"cpe:1"}},
					},
				},
			}, nil).
			AnyTimes()

		u := NewUpdater(g)
		_, err := u.Get(context.Background())
		require.NoError(t, err)

		u.Close()

		mf, err := u.Get(context.Background())
		require.NoError(t, err)
		require.NotNil(t, mf)
		assert.Len(t, mf.Data, 1)
	})

	t.Run("Get after Close without init returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)

		u := NewUpdater(g)
		u.Close()

		mf, err := u.Get(context.Background())
		assert.ErrorIs(t, err, errNoSuccessfulFetch)
		assert.Nil(t, mf)
	})
}
