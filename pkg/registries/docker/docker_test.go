package docker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryMatch(t *testing.T) {
	img, _, err := utils.GenerateImageNameFromString("example.com/remote/repo:tag")
	require.NoError(t, err)

	tt := []struct {
		registry        string
		repoList        set.Set[string]
		disableRepoList bool
		expected        bool
	}{
		// repo list disabled
		{"example.com", nil, true, true},
		{"example.com", set.NewStringSet(), true, true},
		{"example.com", set.NewStringSet("remote/repo"), true, true},

		// repo list enabled
		{"example.com", nil, false, true},
		{"example.com", set.NewStringSet(), false, false},
		{"example.com", set.NewStringSet("remote/repo"), false, true},

		// repeat above with mismatched host
		{"not.example.com", nil, true, false},
		{"not.example.com", set.NewStringSet(), true, false},
		{"not.example.com", set.NewStringSet("remote/repo"), true, false},

		{"not.example.com", nil, false, false},
		{"not.example.com", set.NewStringSet(), false, false},
		{"not.example.com", set.NewStringSet("remote/repo"), false, false},
	}

	for i, tc := range tt {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			r := &Registry{
				registry:             tc.registry,
				repositoryList:       tc.repoList,
				repositoryListTicker: time.NewTicker(repoListInterval),
				cfg:                  &Config{DisableRepoList: tc.disableRepoList},
			}

			// Prevent lazy load from attempting to change repo list under test.
			r.repoListOnce.Do(func() {})

			match := r.Match(img)
			assert.Equal(t, tc.expected, match)
		})
	}
}

func TestLazyLoadRepoList(t *testing.T) {
	var repoListCalls int
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/_catalog" {
			repoListCalls++
			_, _ = w.Write([]byte(`{"repositories":["repo/path"]}`))
			return
		}
	}))

	c := &Config{
		Endpoint:        s.URL,
		DisableRepoList: false,
	}

	hostOnly := urlfmt.TrimHTTPPrefixes(s.URL)

	r, err := NewDockerRegistryWithConfig(c, &storage.ImageIntegration{})
	require.NoError(t, err)
	assert.Zero(t, repoListCalls) // No repo list calls should have been made during construction.

	imgStr := fmt.Sprintf("%s/repo/path:latest", hostOnly)
	imgName, _, err := utils.GenerateImageNameFromString(imgStr)
	require.NoError(t, err)
	assert.True(t, r.Match(imgName))
	assert.Equal(t, 1, repoListCalls) // Lazy load should have executed once.

	imgStr = fmt.Sprintf("%s/no/match:latest", hostOnly)
	imgName, _, err = utils.GenerateImageNameFromString(imgStr)
	require.NoError(t, err)
	assert.False(t, r.Match(imgName))
	assert.Equal(t, 1, repoListCalls) // Lazy load should NOT have executed again.
}

func TestListTags(t *testing.T) {
	tests := []struct {
		name     string
		mockTags []string
	}{
		{
			name: "mixed tags with artifacts",
			mockTags: []string{
				"8.10-1234",
				"8.10-1234.sig",
				"8.10-1234.sbom",
				"8.10-1234.att",
				"8.9-5678",
				"sha256-abc123def456",
				"latest",
			},
		},
		{
			name: "multiple tags",
			mockTags: []string{
				"1.0.0",
				"1.1.0",
				"latest",
				"main",
			},
		},
		{
			name:     "empty tag list",
			mockTags: []string{},
		},
		{
			name: "realistic Red Hat UBI pattern",
			mockTags: []string{
				"8.10-1234",
				"8.10-1234.sig",
				"8.10-1234.sbom",
				"8.10-1234.att",
				"8.10-5678",
				"8.10-5678.sig",
				"8.10-5678.sbom",
				"8.10-5678.att",
				"sha256-aaaa",
				"sha256-bbbb",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP server that returns our test tags
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v2/" {
					// Ping endpoint
					w.WriteHeader(http.StatusOK)
					return
				}
				if r.URL.Path == "/v2/test/repo/tags/list" {
					// Tags list endpoint
					response := `{"name":"test/repo","tags":[`
					for i, tag := range tt.mockTags {
						if i > 0 {
							response += ","
						}
						response += `"` + tag + `"`
					}
					response += `]}`
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(response))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer s.Close()

			// Create registry with mock server
			c := &Config{
				Endpoint:        s.URL,
				DisableRepoList: true,
			}
			r, err := NewDockerRegistryWithConfig(c, &storage.ImageIntegration{})
			require.NoError(t, err)

			// Call ListTags
			tags, err := r.ListTags(t.Context(), "test/repo")
			require.NoError(t, err)

			// Verify all tags are returned
			assert.ElementsMatch(t, tt.mockTags, tags,
				"Expected all tags to be returned")
		})
	}
}

func TestListTagsPagination(t *testing.T) {
	// Test that pagination works by simulating >100 tags
	var allTags []string
	for i := 1; i <= 150; i++ {
		allTags = append(allTags, fmt.Sprintf("tag-%d", i))
	}

	pageSize := 100
	callCount := 0

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/v2/test/repo/tags/list" {
			callCount++

			// Determine which page we're on based on the 'last' parameter
			lastTag := r.URL.Query().Get("last")
			startIdx := 0
			if lastTag != "" {
				// Find the index of the last tag
				for i, tag := range allTags {
					if tag == lastTag {
						startIdx = i + 1
						break
					}
				}
			}

			// Return up to pageSize tags
			endIdx := startIdx + pageSize
			if endIdx > len(allTags) {
				endIdx = len(allTags)
			}
			pageTags := allTags[startIdx:endIdx]

			// Build JSON response
			response := `{"name":"test/repo","tags":[`
			for i, tag := range pageTags {
				if i > 0 {
					response += ","
				}
				response += `"` + tag + `"`
			}
			response += `]}`

			// Add Link header for pagination if there are more tags
			if endIdx < len(allTags) {
				linkHeader := fmt.Sprintf(`</v2/test/repo/tags/list?n=%d&last=%s>; rel="next"`,
					pageSize, pageTags[len(pageTags)-1])
				w.Header().Set("Link", linkHeader)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	c := &Config{
		Endpoint:        s.URL,
		DisableRepoList: true,
	}
	r, err := NewDockerRegistryWithConfig(c, &storage.ImageIntegration{})
	require.NoError(t, err)

	tags, err := r.ListTags(t.Context(), "test/repo")
	require.NoError(t, err)

	// Verify all 150 tags were retrieved (proving pagination works)
	assert.Len(t, tags, 150, "Should retrieve all tags across multiple pages")
	assert.ElementsMatch(t, allTags, tags, "All tags should match")

	// Verify multiple API calls were made (proving pagination occurred)
	assert.Greater(t, callCount, 1, "Should have made multiple paginated calls")
}

func TestListTagsError(t *testing.T) {
	// Test error handling when registry returns an error
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/v2/test/repo/tags/list" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[{"code":"NAME_UNKNOWN","message":"repository not found"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	c := &Config{
		Endpoint:        s.URL,
		DisableRepoList: true,
	}
	r, err := NewDockerRegistryWithConfig(c, &storage.ImageIntegration{})
	require.NoError(t, err)

	tags, err := r.ListTags(t.Context(), "test/repo")
	assert.Error(t, err, "Should return error when repository not found")
	assert.Nil(t, tags, "Should return nil tags on error")
	assert.Contains(t, err.Error(), "failed to list tags", "Error should mention tag listing failure")
}
