package manifest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMetadataStore implements a mock for datastore.IndexerMetadataStore.
type MockMetadataStore struct {
	GCManifestsFunc func(ctx context.Context, expiration time.Time, limit int) ([]string, error)
}

func (m *MockMetadataStore) StoreManifest(ctx context.Context, manifestID string, expiration time.Time) error {
	return nil
}

func (m *MockMetadataStore) ManifestExists(ctx context.Context, manifestID string) (bool, error) {
	return false, nil
}

func (m *MockMetadataStore) GCManifests(ctx context.Context, expiration time.Time, limit int) ([]string, error) {
	if m.GCManifestsFunc != nil {
		return m.GCManifestsFunc(ctx, expiration, limit)
	}
	return nil, nil
}

// MockClairClient implements a mock for clairclient.Client.
type MockClairClient struct {
	DeleteIndexReportFunc func(ctx context.Context, digest string) error
}

func (m *MockClairClient) DeleteIndexReport(ctx context.Context, digest string) error {
	if m.DeleteIndexReportFunc != nil {
		return m.DeleteIndexReportFunc(ctx, digest)
	}
	return nil
}

func TestManager_GC(t *testing.T) {
	tests := map[string]struct {
		expiredManifests []string
		gcError          error
		deleteErrors     map[string]error
		wantDeleteCalls  []string
	}{
		"no expired manifests": {
			expiredManifests: nil,
			wantDeleteCalls:  nil,
		},
		"single expired manifest": {
			expiredManifests: []string{"sha256:abc123"},
			wantDeleteCalls:  []string{"sha256:abc123"},
		},
		"multiple expired manifests": {
			expiredManifests: []string{"sha256:abc123", "sha256:def456", "sha256:ghi789"},
			wantDeleteCalls:  []string{"sha256:abc123", "sha256:def456", "sha256:ghi789"},
		},
		"GC error returns error": {
			gcError:         errors.New("database error"),
			wantDeleteCalls: nil,
		},
		"delete error continues to next manifest": {
			expiredManifests: []string{"sha256:abc123", "sha256:def456", "sha256:ghi789"},
			deleteErrors: map[string]error{
				"sha256:def456": errors.New("delete failed"),
			},
			wantDeleteCalls: []string{"sha256:abc123", "sha256:def456", "sha256:ghi789"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()

			var deleteCalls []string
			mockStore := &MockMetadataStore{
				GCManifestsFunc: func(ctx context.Context, expiration time.Time, limit int) ([]string, error) {
					if tt.gcError != nil {
						return nil, tt.gcError
					}
					assert.Equal(t, 100, limit, "expected default GC throttle")
					return tt.expiredManifests, nil
				},
			}

			mockClair := &MockClairClient{
				DeleteIndexReportFunc: func(ctx context.Context, digest string) error {
					deleteCalls = append(deleteCalls, digest)
					if tt.deleteErrors != nil {
						if err, ok := tt.deleteErrors[digest]; ok {
							return err
						}
					}
					return nil
				},
			}

			m := NewManager(mockStore, mockClair)

			err := m.runGC(ctx)

			if tt.gcError != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get expired manifests")
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantDeleteCalls, deleteCalls)
		})
	}
}

func TestManager_Options(t *testing.T) {
	mockStore := &MockMetadataStore{}
	mockClair := &MockClairClient{}

	t.Run("default options", func(t *testing.T) {
		m := NewManager(mockStore, mockClair)
		assert.Equal(t, time.Hour, m.gcInterval)
		assert.Equal(t, 100, m.gcThrottle)
	})

	t.Run("custom GC interval", func(t *testing.T) {
		m := NewManager(mockStore, mockClair, WithGCInterval(30*time.Minute))
		assert.Equal(t, 30*time.Minute, m.gcInterval)
	})

	t.Run("custom GC throttle", func(t *testing.T) {
		m := NewManager(mockStore, mockClair, WithGCThrottle(50))
		assert.Equal(t, 50, m.gcThrottle)
	})
}

func TestManager_StartStopGC(t *testing.T) {
	ctx := t.Context()

	gcCount := 0
	mockStore := &MockMetadataStore{
		GCManifestsFunc: func(ctx context.Context, expiration time.Time, limit int) ([]string, error) {
			gcCount++
			return nil, nil
		},
	}
	mockClair := &MockClairClient{}

	// Use a very short interval for testing
	m := NewManager(mockStore, mockClair, WithGCInterval(10*time.Millisecond))

	// Start GC in background
	gcCtx, cancel := context.WithCancel(ctx)
	errChan := make(chan error, 1)
	go func() {
		errChan <- m.StartGC(gcCtx)
	}()

	// Wait for at least 2 GC cycles
	time.Sleep(25 * time.Millisecond)

	// Stop GC
	m.StopGC()
	cancel()

	// Wait for GC to finish
	err := <-errChan
	require.NoError(t, err)

	// Should have run GC at least twice (immediate + at least one interval)
	assert.GreaterOrEqual(t, gcCount, 2)
}

func TestManager_StartGC_ContextCancellation(t *testing.T) {
	ctx := t.Context()

	mockStore := &MockMetadataStore{
		GCManifestsFunc: func(ctx context.Context, expiration time.Time, limit int) ([]string, error) {
			return nil, nil
		},
	}
	mockClair := &MockClairClient{}

	m := NewManager(mockStore, mockClair, WithGCInterval(time.Hour))

	// Create a context that we'll cancel immediately
	gcCtx, cancel := context.WithCancel(ctx)

	// Start GC
	errChan := make(chan error, 1)
	go func() {
		errChan <- m.StartGC(gcCtx)
	}()

	// Give it time to start and run first GC
	time.Sleep(10 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for GC to finish
	err := <-errChan
	require.NoError(t, err)
}
