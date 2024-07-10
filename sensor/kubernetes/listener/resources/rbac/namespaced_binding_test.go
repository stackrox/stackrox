//go:build test_all

package rbac

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_namespacedBinding_Equal(t *testing.T) {
	tests := []struct {
		this, that *namespacedBinding
		equal      bool
	}{
		{equal: true},
		{this: &namespacedBinding{}, equal: false},
		{this: &namespacedBinding{}, that: &namespacedBinding{}, equal: true},
		{this: &namespacedBinding{roleRef: namespacedRoleRef{namespace: "test"}}, that: &namespacedBinding{}, equal: false},
		{this: &namespacedBinding{roleRef: namespacedRoleRef{namespace: "test"}}, that: &namespacedBinding{roleRef: namespacedRoleRef{namespace: "test"}}, equal: true},
		{this: &namespacedBinding{
			roleRef:  namespacedRoleRef{namespace: "test"},
			subjects: []namespacedSubject{},
		}, that: &namespacedBinding{roleRef: namespacedRoleRef{namespace: "test"}}, equal: true},
		{this: &namespacedBinding{
			roleRef:  namespacedRoleRef{namespace: "test"},
			subjects: []namespacedSubject{"a", "b"},
		}, that: &namespacedBinding{roleRef: namespacedRoleRef{namespace: "test"}}, equal: false},
		{this: &namespacedBinding{
			roleRef:  namespacedRoleRef{namespace: "test"},
			subjects: []namespacedSubject{"a", "b"},
		}, that: &namespacedBinding{
			roleRef:  namespacedRoleRef{namespace: "test"},
			subjects: []namespacedSubject{"a", "b"},
		}, equal: true},
		{this: &namespacedBinding{
			roleRef:  namespacedRoleRef{namespace: "test"},
			subjects: []namespacedSubject{"a", "b"},
		}, that: &namespacedBinding{
			roleRef:  namespacedRoleRef{namespace: "test"},
			subjects: []namespacedSubject{"b", "a"},
		}, equal: true},
		{this: &namespacedBinding{
			roleRef:  namespacedRoleRef{namespace: "test"},
			subjects: []namespacedSubject{"a", "b", "d"},
		}, that: &namespacedBinding{
			roleRef:  namespacedRoleRef{namespace: "test"},
			subjects: []namespacedSubject{"b", "a", "c"},
		}, equal: false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("(%+v == %+v) == %t", tt.this, tt.that, tt.equal), func(t *testing.T) {
			assert.Equal(t, tt.equal, tt.this.Equal(tt.that))
			assert.Equal(t, tt.equal, tt.that.Equal(tt.this))
		})
	}
}
