package datastore

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func makeProfile(name, clusterID string, kind storage.ComplianceOperatorProfileV2_OperatorKind, hash string) *storage.ComplianceOperatorProfileV2 {
	return &storage.ComplianceOperatorProfileV2{
		Name:            name,
		ClusterId:       clusterID,
		OperatorKind:    kind,
		EquivalenceHash: hash,
	}
}

const (
	standard = storage.ComplianceOperatorProfileV2_PROFILE
	tailored = storage.ComplianceOperatorProfileV2_TAILORED_PROFILE
)

// buildByName is a test helper that groups a profile list the same way filterNonEquivalentTPs does.
func buildByName(profiles []*storage.ComplianceOperatorProfileV2) map[string][]*storage.ComplianceOperatorProfileV2 {
	byName := make(map[string][]*storage.ComplianceOperatorProfileV2)
	for _, p := range profiles {
		byName[p.GetName()] = append(byName[p.GetName()], p)
	}
	return byName
}

func TestApplyEquivalenceFilter_OOBPassesThrough(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("ocp4-cis", "c1", standard, ""),
		makeProfile("ocp4-cis", "c2", standard, ""),
	}
	names := []string{"ocp4-cis"}
	got := applyEquivalenceFilter(names, buildByName(profiles))
	assert.Equal(t, []string{"ocp4-cis"}, got)
}

func TestApplyEquivalenceFilter_TPSameHashPassesThrough(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("my-tp", "c1", tailored, "hash-abc"),
		makeProfile("my-tp", "c2", tailored, "hash-abc"),
	}
	names := []string{"my-tp"}
	got := applyEquivalenceFilter(names, buildByName(profiles))
	assert.Equal(t, []string{"my-tp"}, got)
}

func TestApplyEquivalenceFilter_TPDifferentHashExcluded(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("my-tp", "c1", tailored, "hash-abc"),
		makeProfile("my-tp", "c2", tailored, "hash-xyz"),
	}
	names := []string{"my-tp"}
	got := applyEquivalenceFilter(names, buildByName(profiles))
	assert.Empty(t, got)
}

func TestApplyEquivalenceFilter_AllEmptyHashEquivalent(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("my-tp", "c1", tailored, ""),
		makeProfile("my-tp", "c2", tailored, ""),
	}
	names := []string{"my-tp"}
	got := applyEquivalenceFilter(names, buildByName(profiles))
	assert.Equal(t, []string{"my-tp"}, got)
}

func TestApplyEquivalenceFilter_PreservesOrder(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("tp-a", "c1", tailored, "h"),
		makeProfile("tp-a", "c2", tailored, "h"),
		makeProfile("tp-bad", "c1", tailored, "h1"),
		makeProfile("tp-bad", "c2", tailored, "h2"),
		makeProfile("ocp4-cis", "c1", standard, ""),
		makeProfile("ocp4-cis", "c2", standard, ""),
	}
	names := []string{"tp-a", "tp-bad", "ocp4-cis"}
	got := applyEquivalenceFilter(names, buildByName(profiles))
	assert.Equal(t, []string{"tp-a", "ocp4-cis"}, got)
}

func TestApplyEquivalenceFilter_EmptyInput(t *testing.T) {
	got := applyEquivalenceFilter(nil, nil)
	assert.Nil(t, got)
}

func TestTailoredProfilesEquivalent(t *testing.T) {
	tests := []struct {
		name      string
		instances []*storage.ComplianceOperatorProfileV2
		want      bool
	}{
		{
			name:      "nil slice",
			instances: nil,
			want:      true,
		},
		{
			name:      "empty slice",
			instances: []*storage.ComplianceOperatorProfileV2{},
			want:      true,
		},
		{
			name:      "single instance",
			instances: []*storage.ComplianceOperatorProfileV2{makeProfile("x", "c1", tailored, "abc")},
			want:      true,
		},
		{
			name: "same hash across clusters",
			instances: []*storage.ComplianceOperatorProfileV2{
				makeProfile("x", "c1", tailored, "abc"),
				makeProfile("x", "c2", tailored, "abc"),
			},
			want: true,
		},
		{
			name: "all-empty hash treated as equivalent",
			instances: []*storage.ComplianceOperatorProfileV2{
				makeProfile("x", "c1", tailored, ""),
				makeProfile("x", "c2", tailored, ""),
			},
			want: true,
		},
		{
			name: "different hashes",
			instances: []*storage.ComplianceOperatorProfileV2{
				makeProfile("x", "c1", tailored, "abc"),
				makeProfile("x", "c2", tailored, "xyz"),
			},
			want: false,
		},
		{
			name: "one empty one non-empty hash",
			instances: []*storage.ComplianceOperatorProfileV2{
				makeProfile("x", "c1", tailored, "abc"),
				makeProfile("x", "c2", tailored, ""),
			},
			want: false,
		},
		{
			name: "three clusters, all same hash",
			instances: []*storage.ComplianceOperatorProfileV2{
				makeProfile("x", "c1", tailored, "h"),
				makeProfile("x", "c2", tailored, "h"),
				makeProfile("x", "c3", tailored, "h"),
			},
			want: true,
		},
		{
			name: "three clusters, last differs",
			instances: []*storage.ComplianceOperatorProfileV2{
				makeProfile("x", "c1", tailored, "h"),
				makeProfile("x", "c2", tailored, "h"),
				makeProfile("x", "c3", tailored, "z"),
			},
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, TailoredProfilesEquivalent(tc.instances))
		})
	}
}
