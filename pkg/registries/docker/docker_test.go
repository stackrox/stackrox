package docker

import (
	"fmt"
	"testing"
	"time"

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
				cfg:                  Config{DisableRepoList: tc.disableRepoList},
			}

			match := r.Match(img)
			assert.Equal(t, tc.expected, match)
		})
	}
}
