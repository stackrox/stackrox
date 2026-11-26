package authn

import (
	"testing"

	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
)

func TestFilterOutNoneRole(t *testing.T) {
	noneRole1 := roletest.NewResolvedRole(NoneRole, nil, nil)
	noneRole2 := roletest.NewResolvedRole(NoneRole, nil, nil)
	otherRole1 := roletest.NewResolvedRole("Role1", nil, nil)
	otherRole2 := roletest.NewResolvedRole("Role2", nil, nil)

	for name, tc := range map[string]struct {
		roles    []permissions.ResolvedRole
		expected []permissions.ResolvedRole
	}{
		"nil input yields empty slice": {
			roles:    nil,
			expected: []permissions.ResolvedRole{},
		},
		"empty input yields empty slice": {
			roles:    []permissions.ResolvedRole{},
			expected: []permissions.ResolvedRole{},
		},
		"None roles are filtered out": {
			roles:    []permissions.ResolvedRole{noneRole1, noneRole2},
			expected: []permissions.ResolvedRole{},
		},
		"nil roles are filtered out": {
			roles:    []permissions.ResolvedRole{nil, otherRole1},
			expected: []permissions.ResolvedRole{otherRole1},
		},
		"non-None roles are preserved in order": {
			roles:    []permissions.ResolvedRole{noneRole1, otherRole1, noneRole2, otherRole2},
			expected: []permissions.ResolvedRole{otherRole1, otherRole2},
		},
	} {
		t.Run(name, func(it *testing.T) {
			assert.Equal(it, tc.expected, FilterOutNoneRole(tc.roles))
		})
	}
}
