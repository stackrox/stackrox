package reposcan

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	registryTypesMocks "github.com/stackrox/rox/pkg/registries/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListAndFilterTags_Matching(t *testing.T) {
	testCases := []struct {
		name     string
		allTags  []string
		pattern  string
		expected []string
	}{
		{
			name:     "star wildcard",
			allTags:  []string{"1.0", "1.1", "2.0", "latest"},
			pattern:  "1.*",
			expected: []string{"1.0", "1.1"},
		},
		{
			name:     "suffix pattern",
			allTags:  []string{"v1.0", "v2.0", "latest", "v3.0-beta"},
			pattern:  "v*",
			expected: []string{"v1.0", "v2.0", "v3.0-beta"},
		},
		{
			name:     "exact match",
			allTags:  []string{"latest", "v1.0", "stable"},
			pattern:  "latest",
			expected: []string{"latest"},
		},
		{
			name:     "question mark",
			allTags:  []string{"v1", "v2", "v10", "v20"},
			pattern:  "v?",
			expected: []string{"v1", "v2"},
		},
		{
			name:     "character class",
			allTags:  []string{"v1.0", "v2.0", "v3.0", "latest"},
			pattern:  "v[12].*",
			expected: []string{"v1.0", "v2.0"},
		},
		{
			name:     "negated character class",
			allTags:  []string{"v1.0", "v2.0", "va.0", "vb.0"},
			pattern:  "v[^12].*",
			expected: []string{"va.0", "vb.0"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRegistry := registryTypesMocks.NewMockRegistry(ctrl)

			mockRegistry.EXPECT().
				ListTags(gomock.Any(), "test/repo").
				Return(tc.allTags, nil)

			result, err := ListAndFilterTags(context.Background(), mockRegistry, "test/repo", tc.pattern)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestListAndFilterTags_NoMatches(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistry := registryTypesMocks.NewMockRegistry(ctrl)

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), "test/repo").
		Return([]string{"v1.0", "v2.0", "latest"}, nil)

	result, err := ListAndFilterTags(context.Background(), mockRegistry, "test/repo", "nonexistent*")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestListAndFilterTags_EmptyPattern(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistry := registryTypesMocks.NewMockRegistry(ctrl)

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), "test/repo").
		Return([]string{"v1.0", "v2.0"}, nil)

	result, err := ListAndFilterTags(context.Background(), mockRegistry, "test/repo", "")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestListAndFilterTags_RegistryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistry := registryTypesMocks.NewMockRegistry(ctrl)

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), "test/repo").
		Return(nil, errox.InvariantViolation.New("connection refused"))

	result, err := ListAndFilterTags(context.Background(), mockRegistry, "test/repo", "*")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "list tags failed")
}

func TestListAndFilterTags_InvalidPattern(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistry := registryTypesMocks.NewMockRegistry(ctrl)

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), "test/repo").
		Return([]string{"v1.0"}, nil)

	// "[" is an invalid glob pattern (unclosed bracket).
	result, err := ListAndFilterTags(context.Background(), mockRegistry, "test/repo", "[")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid glob pattern")
}

func TestListAndFilterTags_RealWorldPatterns(t *testing.T) {
	testCases := []struct {
		name     string
		allTags  []string
		pattern  string
		expected []string
	}{
		{
			name:     "UBI 8.x versions",
			allTags:  []string{"8.0", "8.1", "8.2", "8.10", "9.0", "9.1", "latest"},
			pattern:  "8.*",
			expected: []string{"8.0", "8.1", "8.2", "8.10"},
		},
		{
			name:     "UBI 8.10 only",
			allTags:  []string{"8.0", "8.1", "8.10", "8.10-123", "8.10-456"},
			pattern:  "8.10*",
			expected: []string{"8.10", "8.10-123", "8.10-456"},
		},
		{
			name:     "latest tag only",
			allTags:  []string{"v1.0", "v2.0", "latest", "stable"},
			pattern:  "latest",
			expected: []string{"latest"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRegistry := registryTypesMocks.NewMockRegistry(ctrl)

			mockRegistry.EXPECT().
				ListTags(gomock.Any(), "ubi8/ubi").
				Return(tc.allTags, nil)

			result, err := ListAndFilterTags(context.Background(), mockRegistry, "ubi8/ubi", tc.pattern)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestListAndFilterTags_LargeTagList(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistry := registryTypesMocks.NewMockRegistry(ctrl)

	// Generate 1000 tags.
	allTags := make([]string, 1000)
	for i := range allTags {
		if i < 100 {
			allTags[i] = "v1." + string(rune('0'+i%10)) + string(rune('0'+i/10%10))
		} else {
			allTags[i] = "v2." + string(rune('0'+i%10))
		}
	}

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), "test/repo").
		Return(allTags, nil)

	result, err := ListAndFilterTags(context.Background(), mockRegistry, "test/repo", "v1.*")
	require.NoError(t, err)
	assert.Len(t, result, 100)
}
