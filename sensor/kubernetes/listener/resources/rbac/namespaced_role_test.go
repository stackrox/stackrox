package rbac

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func Test_namespacedRole_Equal(t *testing.T) {
	tests := []struct {
		this, that *namespacedRole
		equal      bool
	}{
		{equal: true},
		{this: &namespacedRole{}, that: &namespacedRole{}, equal: true},
		{this: &namespacedRole{latestUID: "a"}, that: &namespacedRole{latestUID: "a"}, equal: true},
		{this: &namespacedRole{latestUID: "a", rules: []*storage.PolicyRule{}}, that: &namespacedRole{latestUID: "a"}, equal: true},
		{
			this: &namespacedRole{latestUID: "a", rules: []*storage.PolicyRule{
				{Verbs: []string{"a", "b"}}, {Verbs: []string{"c", "d"}},
			}},
			that:  &namespacedRole{latestUID: "a"},
			equal: false,
		},
		{
			this: &namespacedRole{latestUID: "a", rules: []*storage.PolicyRule{
				{Verbs: []string{"a", "b"}}, {Verbs: []string{"c", "d"}},
			}},
			that: &namespacedRole{latestUID: "a", rules: []*storage.PolicyRule{
				{Verbs: []string{"a", "b"}}, {Verbs: []string{"c", "d"}},
			}},
			equal: true,
		},
		{
			this: &namespacedRole{latestUID: "a", rules: []*storage.PolicyRule{
				{Verbs: []string{"b", "a"}}, {Verbs: []string{"c", "d"}},
			}},
			that: &namespacedRole{latestUID: "a", rules: []*storage.PolicyRule{
				{Verbs: []string{"a", "b"}}, {Verbs: []string{"c", "d"}},
			}},
			equal: false,
		},
		{
			this: &namespacedRole{latestUID: "a", rules: []*storage.PolicyRule{
				{Verbs: []string{"a", "b"}}, {Verbs: []string{"c", "d"}},
			}},
			that: &namespacedRole{latestUID: "a", rules: []*storage.PolicyRule{
				{Verbs: []string{"c", "d"}}, {Verbs: []string{"a", "b"}},
			}},
			equal: false,
		},
		{this: &namespacedRole{latestUID: "a"}, that: &namespacedRole{latestUID: ""}, equal: false},
		{this: &namespacedRole{latestUID: "a"}, equal: false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("(%+v == %+v) == %t", tt.this, tt.that, tt.equal), func(t *testing.T) {
			assert.Equal(t, tt.equal, tt.this.Equal(tt.that))
			assert.Equal(t, tt.equal, tt.that.Equal(tt.this))
		})
	}
}
