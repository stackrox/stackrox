package watcher

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRegistry is a test implementation of types.Registry for testing
type mockRegistry struct {
	tags      []string
	err       error
	callCount int
}

func (m *mockRegistry) ListTags(_ context.Context, _ string) ([]string, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.tags, nil
}

// Stub implementations for required Registry interface methods
func (m *mockRegistry) Match(_ *storage.ImageName) bool                           { return false }
func (m *mockRegistry) Metadata(_ *storage.Image) (*storage.ImageMetadata, error) { return nil, nil }
func (m *mockRegistry) Test() error                                               { return nil }
func (m *mockRegistry) Config(_ context.Context) *types.Config                    { return nil }
func (m *mockRegistry) Name() string                                              { return "mock" }
func (m *mockRegistry) HTTPClient() *http.Client                                  { return nil }

func TestListAndFilterTags_Matching(t *testing.T) {
	tests := []struct {
		name     string
		allTags  []string
		pattern  string
		wantTags []string
	}{
		{
			name:     "star wildcard",
			allTags:  []string{"8.1", "8.2", "8.10", "7.9", "9.0"},
			pattern:  "8.*",
			wantTags: []string{"8.1", "8.2", "8.10"},
		},
		{
			name:     "suffix pattern",
			allTags:  []string{"3.14-alpine", "3.14-ubuntu", "latest-alpine", "stable"},
			pattern:  "*-alpine",
			wantTags: []string{"3.14-alpine", "latest-alpine"},
		},
		{
			name:     "exact match",
			allTags:  []string{"latest", "stable", "edge", "latest-alpine"},
			pattern:  "latest",
			wantTags: []string{"latest"},
		},
		{
			name:     "question mark",
			allTags:  []string{"v1.2.0", "v1.2.1", "v1.2.10", "v1.3.0"},
			pattern:  "v1.2.?",
			wantTags: []string{"v1.2.0", "v1.2.1"},
		},
		{
			name:     "character class",
			allTags:  []string{"v1.0", "v2.0", "v3.0", "va.0"},
			pattern:  "v[12].*",
			wantTags: []string{"v1.0", "v2.0"},
		},
		{
			name:     "negated character class",
			allTags:  []string{"v1.0", "v2.0", "va.0", "vb.0"},
			pattern:  "v[^12].*",
			wantTags: []string{"va.0", "vb.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRegistry{tags: tt.allTags}
			ctx := context.Background()

			gotTags, err := listAndFilterTags(ctx, mock, "test/repo", tt.pattern)

			require.NoError(t, err)
			assert.ElementsMatch(t, tt.wantTags, gotTags)
			assert.Equal(t, 1, mock.callCount, "ListTags should be called exactly once")
		})
	}
}

func TestListAndFilterTags_NoMatches(t *testing.T) {
	mock := &mockRegistry{
		tags: []string{"1.0", "2.0", "3.0"},
	}
	ctx := context.Background()

	gotTags, err := listAndFilterTags(ctx, mock, "test/repo", "9.*")

	require.NoError(t, err, "no matches should not return error")
	assert.Empty(t, gotTags, "should return empty slice when no matches")
}

func TestListAndFilterTags_EmptyPattern(t *testing.T) {
	allTags := []string{"8.1", "8.2", "8.10", "latest", "stable"}
	mock := &mockRegistry{tags: allTags}
	ctx := context.Background()

	gotTags, err := listAndFilterTags(ctx, mock, "test/repo", "")

	require.NoError(t, err, "empty pattern should not error")
	assert.Empty(t, gotTags, "empty pattern matches no tags")
}

func TestListAndFilterTags_RegistryError(t *testing.T) {
	registryErr := errors.New("connection timeout")
	mock := &mockRegistry{err: registryErr}
	ctx := context.Background()

	gotTags, err := listAndFilterTags(ctx, mock, "test/repo", "8.*")

	require.Error(t, err, "should propagate registry error")
	assert.Nil(t, gotTags, "should return nil tags on error")
	assert.Contains(t, err.Error(), "connection timeout")
	assert.Contains(t, err.Error(), "list tags failed")
}

func TestListAndFilterTags_InvalidPattern(t *testing.T) {
	mock := &mockRegistry{tags: []string{"tag1", "tag2"}}
	ctx := context.Background()

	// Malformed pattern (unclosed bracket)
	_, err := listAndFilterTags(ctx, mock, "test/repo", "[invalid")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid glob pattern")
}

func TestListAndFilterTags_RealWorldPatterns(t *testing.T) {
	allTags := []string{
		"8.1-1",
		"8.1-2",
		"8.2-1",
		"8.10-1234",
		"8.10-5678",
		"9.0-1",
		"latest",
		"8.1-1.1234567890",
		"8.10-1234.5678901234",
	}

	tests := []struct {
		name     string
		pattern  string
		wantTags []string
	}{
		{
			name:     "UBI 8.x versions",
			pattern:  "8.*",
			wantTags: []string{"8.1-1", "8.1-2", "8.2-1", "8.10-1234", "8.10-5678", "8.1-1.1234567890", "8.10-1234.5678901234"},
		},
		{
			name:     "UBI 8.10 only",
			pattern:  "8.10-*",
			wantTags: []string{"8.10-1234", "8.10-5678", "8.10-1234.5678901234"},
		},
		{
			name:     "latest tag only",
			pattern:  "latest",
			wantTags: []string{"latest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRegistry{tags: allTags}
			ctx := context.Background()

			gotTags, err := listAndFilterTags(ctx, mock, "registry.redhat.io/ubi8/ubi", tt.pattern)

			require.NoError(t, err)
			assert.ElementsMatch(t, tt.wantTags, gotTags)
		})
	}
}

func TestListAndFilterTags_LargeTagList(t *testing.T) {
	// Verify efficiency with large tag lists
	var allTags []string
	for i := 0; i < 1000; i++ {
		allTags = append(allTags, fmt.Sprintf("tag-%d", i))
	}
	for i := 0; i < 100; i++ {
		allTags = append(allTags, fmt.Sprintf("match-%d", i))
	}

	mock := &mockRegistry{tags: allTags}
	ctx := context.Background()

	gotTags, err := listAndFilterTags(ctx, mock, "test/repo", "match-*")

	require.NoError(t, err)
	assert.Len(t, gotTags, 100)
}
