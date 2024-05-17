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

			// Prevent lazy load from attempt to change repo list.
			r.repoListOnce.Do(func() {})

			match := r.Match(img)
			assert.Equal(t, tc.expected, match)
		})
	}
}

func TestLazyLoadRepoList(t *testing.T) {
	numRepoListCalls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/_catalog" {
			numRepoListCalls++
			return
		}
	}))

	c := &Config{
		Endpoint:        s.URL,
		DisableRepoList: false,
	}

	r, err := NewDockerRegistryWithConfig(c, &storage.ImageIntegration{})
	require.NoError(t, err)

	// No calls should have been made to repo during construction.
	assert.Zero(t, numRepoListCalls)

	r.Match(&storage.ImageName{})

	// A call should now have been made to lazy load repo list.
	assert.Equal(t, 1, numRepoListCalls)

	r.Match(&storage.ImageName{})

	// Ensure only a single attempt was made to load initial repo list.
	assert.Equal(t, 1, numRepoListCalls)
}
