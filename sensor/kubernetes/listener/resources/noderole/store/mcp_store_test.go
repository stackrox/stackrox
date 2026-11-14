package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	tests := map[string]struct {
		existing []string
		add      string
		want     bool
	}{
		"add to empty store": {
			add:  "master",
			want: true,
		},
		"add new name": {
			existing: []string{"master"},
			add:      "worker",
			want:     true,
		},
		"add duplicate": {
			existing: []string{"master"},
			add:      "master",
			want:     false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := NewMachineConfigPoolStore()
			for _, e := range tt.existing {
				s.Add(e)
			}
			got := s.Add(tt.add)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRemove(t *testing.T) {
	tests := map[string]struct {
		existing []string
		remove   string
		want     bool
	}{
		"remove from empty store": {
			remove: "master",
			want:   false,
		},
		"remove existing name": {
			existing: []string{"master", "worker"},
			remove:   "master",
			want:     true,
		},
		"remove non-existing name": {
			existing: []string{"master"},
			remove:   "worker",
			want:     false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := NewMachineConfigPoolStore()
			for _, e := range tt.existing {
				s.Add(e)
			}
			got := s.Remove(tt.remove)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetRoles(t *testing.T) {
	s := NewMachineConfigPoolStore()
	s.Add("master")
	s.Add("worker")
	s.Add("infra")

	roles := s.GetRoles()
	assert.Len(t, roles, 3)
	assert.ElementsMatch(t, []string{"master", "worker", "infra"}, roles)
}

func TestGetRolesReturnsSnapshot(t *testing.T) {
	s := NewMachineConfigPoolStore()
	s.Add("master")

	snapshot := s.GetRoles()
	s.Add("worker")

	assert.Len(t, snapshot, 1, "snapshot should not reflect later mutations")
	assert.ElementsMatch(t, []string{"master", "worker"}, s.GetRoles())
}

func TestCleanup(t *testing.T) {
	s := NewMachineConfigPoolStore()
	s.Add("master")
	s.Add("worker")
	assert.Len(t, s.GetRoles(), 2)

	s.Cleanup()
	assert.Empty(t, s.GetRoles())
}

func TestCleanupThenReuse(t *testing.T) {
	s := NewMachineConfigPoolStore()
	s.Add("master")
	s.Cleanup()

	assert.True(t, s.Add("master"), "should be able to add after cleanup")
	assert.ElementsMatch(t, []string{"master"}, s.GetRoles())
}
