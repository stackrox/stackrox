package store

import (
	"testing"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
)

func TestUpsert(t *testing.T) {
	tests := map[string]struct {
		existing map[string][]string
		name     string
		roles    []string
		want     bool
	}{
		"upsert into empty store": {
			name:  "master",
			roles: []string{"master"},
			want:  true,
		},
		"upsert new pool with new role": {
			existing: map[string][]string{"master": {"master"}},
			name:     "infra",
			roles:    []string{"infra"},
			want:     true,
		},
		"upsert same pool with same roles": {
			existing: map[string][]string{"worker": {"worker"}},
			name:     "worker",
			roles:    []string{"worker"},
			want:     false,
		},
		"upsert pool with changed roles": {
			existing: map[string][]string{"infra": {"worker"}},
			name:     "infra",
			roles:    []string{"infra"},
			want:     true,
		},
		"upsert new pool with already tracked roles": {
			existing: map[string][]string{"worker": {"worker", "infra"}},
			name:     "infra",
			roles:    []string{"infra"},
			want:     false,
		},
		"upsert pool with no roles": {
			name:  "empty",
			roles: nil,
			want:  false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := NewMachineConfigPoolStore()
			for poolName, roles := range tt.existing {
				s.Upsert(poolName, set.NewStringSet(roles...))
			}
			got := s.Upsert(tt.name, set.NewStringSet(tt.roles...))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRemove(t *testing.T) {
	tests := map[string]struct {
		existing map[string][]string
		remove   string
		want     bool
	}{
		"remove from empty store": {
			remove: "master",
			want:   false,
		},
		"remove existing pool with exclusive roles": {
			existing: map[string][]string{"master": {"master"}, "worker": {"worker"}},
			remove:   "master",
			want:     true,
		},
		"remove pool sharing roles with another pool": {
			existing: map[string][]string{"worker": {"worker"}, "infra": {"worker", "infra"}},
			remove:   "infra",
			want:     true,
		},
		"remove pool whose roles are covered by another pool": {
			existing: map[string][]string{"worker": {"worker", "infra"}, "infra": {"infra"}},
			remove:   "infra",
			want:     false,
		},
		"remove non-existent pool": {
			existing: map[string][]string{"master": {"master"}},
			remove:   "worker",
			want:     false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := NewMachineConfigPoolStore()
			for poolName, roles := range tt.existing {
				s.Upsert(poolName, set.NewStringSet(roles...))
			}
			got := s.Remove(tt.remove)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetRoles(t *testing.T) {
	s := NewMachineConfigPoolStore()
	s.Upsert("master", set.NewStringSet("master"))
	s.Upsert("worker", set.NewStringSet("worker"))
	s.Upsert("infra", set.NewStringSet("worker", "infra"))

	roles := s.GetRoles()
	assert.Len(t, roles, 3)
	assert.ElementsMatch(t, []string{"master", "worker", "infra"}, roles)
}

func TestGetRolesReturnsSnapshot(t *testing.T) {
	s := NewMachineConfigPoolStore()
	s.Upsert("master", set.NewStringSet("master"))

	snapshot := s.GetRoles()
	s.Upsert("worker", set.NewStringSet("worker"))

	assert.Len(t, snapshot, 1, "snapshot should not reflect later mutations")
	assert.ElementsMatch(t, []string{"master", "worker"}, s.GetRoles())
}

func TestGetRolesAfterRoleChange(t *testing.T) {
	s := NewMachineConfigPoolStore()
	s.Upsert("infra", set.NewStringSet("worker"))

	s.Upsert("infra", set.NewStringSet("infra"))
	assert.ElementsMatch(t, []string{"infra"}, s.GetRoles())
}

func TestCleanup(t *testing.T) {
	s := NewMachineConfigPoolStore()
	s.Upsert("master", set.NewStringSet("master"))
	s.Upsert("worker", set.NewStringSet("worker"))
	assert.Len(t, s.GetRoles(), 2)

	s.Cleanup()
	assert.Empty(t, s.GetRoles())
}

func TestCleanupThenReuse(t *testing.T) {
	s := NewMachineConfigPoolStore()
	s.Upsert("master", set.NewStringSet("master"))
	s.Cleanup()

	assert.True(t, s.Upsert("master", set.NewStringSet("master")), "should be able to add after cleanup")
	assert.ElementsMatch(t, []string{"master"}, s.GetRoles())
}
