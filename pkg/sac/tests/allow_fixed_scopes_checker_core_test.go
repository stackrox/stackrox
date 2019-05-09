package tests

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	. "github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

func TestAllowFixedScopes(t *testing.T) {
	t.Parallel()

	resA := permissions.Resource("resA")
	resB := permissions.Resource("resB")
	resC := permissions.Resource("resC")

	sc := NewScopeChecker(AllowFixedScopes(
		AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		ResourceScopeKeys(resA, resB),
	))

	cases := []struct {
		scope    []ScopeKey
		expected TryAllowedResult
	}{
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
			},
			expected: Deny,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
			},
			expected: Deny,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resC),
			},
			expected: Deny,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resC),
			},
			expected: Deny,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resA),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resA),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resB),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resB),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resC),
				ClusterScopeKey("someCluster"),
			},
			expected: Deny,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resC),
				ClusterScopeKey("someCluster"),
			},
			expected: Deny,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resA),
				ClusterScopeKey("someCluster"),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resA),
				ClusterScopeKey("someCluster"),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resB),
				ClusterScopeKey("someCluster"),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resB),
				ClusterScopeKey("someCluster"),
			},
			expected: Allow,
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.expected, sc.TryAllowed(c.scope...), "expected result for scope %v to be %s", c.scope, c.expected)
	}
}
