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
	standard      = storage.ComplianceOperatorProfileV2_PROFILE
	tailored = storage.ComplianceOperatorProfileV2_TAILORED_PROFILE
)

func TestResolveEligibleProfileNames_OOBOnAllClusters(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("ocp4-cis", "cluster-1", standard, ""),
		makeProfile("ocp4-cis", "cluster-2", standard, ""),
	}
	tailoredNames, standardNames := resolveEligibleProfileNames(profiles, 2, false)
	assert.Empty(t, tailoredNames)
	assert.Equal(t, []string{"ocp4-cis"}, standardNames)
}

func TestResolveEligibleProfileNames_TPSameHashAllClusters(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("my-tp", "cluster-1", tailored, "hash-abc"),
		makeProfile("my-tp", "cluster-2", tailored, "hash-abc"),
	}
	tailoredNames, standardNames := resolveEligibleProfileNames(profiles, 2, false)
	assert.Equal(t, []string{"my-tp"}, tailoredNames)
	assert.Empty(t, standardNames)
}

func TestResolveEligibleProfileNames_TPDifferentHashExcluded(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("my-tp", "cluster-1", tailored, "hash-abc"),
		makeProfile("my-tp", "cluster-2", tailored, "hash-xyz"),
	}
	tailoredNames, standardNames := resolveEligibleProfileNames(profiles, 2, false)
	assert.Empty(t, tailoredNames)
	assert.Empty(t, standardNames)
}

func TestResolveEligibleProfileNames_MixedKindExcluded(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("mixed", "cluster-1", standard, ""),
		makeProfile("mixed", "cluster-2", tailored, "hash-abc"),
	}
	tailoredNames, standardNames := resolveEligibleProfileNames(profiles, 2, false)
	assert.Empty(t, tailoredNames)
	assert.Empty(t, standardNames)
}

func TestResolveEligibleProfileNames_NotOnAllClustersExcluded(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("ocp4-cis", "cluster-1", standard, ""),
		// missing from cluster-2
	}
	tailoredNames, standardNames := resolveEligibleProfileNames(profiles, 2, false)
	assert.Empty(t, tailoredNames)
	assert.Empty(t, standardNames)
}

func TestResolveEligibleProfileNames_SingleCluster(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("my-tp", "cluster-1", tailored, "hash-abc"),
	}
	tailoredNames, standardNames := resolveEligibleProfileNames(profiles, 1, false)
	assert.Equal(t, []string{"my-tp"}, tailoredNames)
	assert.Empty(t, standardNames)
}

func TestResolveEligibleProfileNames_SkipHashIncludesTPsWithDifferentHashes(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("my-tp", "cluster-1", tailored, "hash-abc"),
		makeProfile("my-tp", "cluster-2", tailored, "hash-xyz"),
	}
	// skipHash=true: different hashes should not exclude the tailored profile
	tailoredNames, standardNames := resolveEligibleProfileNames(profiles, 2, true)
	assert.Equal(t, []string{"my-tp"}, tailoredNames)
	assert.Empty(t, standardNames)
}

func TestResolveEligibleProfileNames_AllEmptyHashEquivalent(t *testing.T) {
	// All-empty hash is treated as equivalent (sensor bug fallback).
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("my-tp", "cluster-1", tailored, ""),
		makeProfile("my-tp", "cluster-2", tailored, ""),
	}
	tailoredNames, standardNames := resolveEligibleProfileNames(profiles, 2, false)
	assert.Equal(t, []string{"my-tp"}, tailoredNames)
	assert.Empty(t, standardNames)
}

func TestGroupProfilesByName(t *testing.T) {
	profiles := []*storage.ComplianceOperatorProfileV2{
		makeProfile("a", "c1", standard, ""),
		makeProfile("a", "c2", standard, ""),
		makeProfile("b", "c1", tailored, "h"),
	}
	got := groupProfilesByName(profiles)
	assert.Len(t, got["a"], 2)
	assert.Len(t, got["b"], 1)
	assert.Len(t, got, 2)
}

func TestRetainPresentOnAllClusters(t *testing.T) {
	input := map[string][]*storage.ComplianceOperatorProfileV2{
		"present-on-both": {makeProfile("present-on-both", "c1", standard, ""), makeProfile("present-on-both", "c2", standard, "")},
		"only-one":        {makeProfile("only-one", "c1", standard, "")},
	}
	got := retainPresentOnAllClusters(input, 2)
	assert.Contains(t, got, "present-on-both")
	assert.NotContains(t, got, "only-one")
}

func TestPartitionByKind(t *testing.T) {
	input := map[string][]*storage.ComplianceOperatorProfileV2{
		"tailored":    {makeProfile("tailored", "c1", tailored, "h"), makeProfile("tailored", "c2", tailored, "h")},
		"standard":   {makeProfile("standard", "c1", standard, ""), makeProfile("standard", "c2", standard, "")},
		"mixed": {makeProfile("mixed", "c1", standard, ""), makeProfile("mixed", "c2", tailored, "h")},
	}
	tailoredGroups, standardGroups := partitionByKind(input)
	assert.Contains(t, tailoredGroups, "tailored")
	assert.Contains(t, standardGroups, "standard")
	assert.NotContains(t, tailoredGroups, "mixed")
	assert.NotContains(t, standardGroups, "mixed")
}

func TestTailoredNamesWithConsistentHash(t *testing.T) {
	input := map[string][]*storage.ComplianceOperatorProfileV2{
		"same-hash": {makeProfile("same-hash", "c1", tailored, "abc"), makeProfile("same-hash", "c2", tailored, "abc")},
		"diff-hash": {makeProfile("diff-hash", "c1", tailored, "abc"), makeProfile("diff-hash", "c2", tailored, "xyz")},
	}
	names := tailoredNamesWithConsistentHash(input, false)
	assert.Contains(t, names, "same-hash")
	assert.NotContains(t, names, "diff-hash")

	// skipHash bypasses the check
	names = tailoredNamesWithConsistentHash(input, true)
	assert.Contains(t, names, "same-hash")
	assert.Contains(t, names, "diff-hash")
}

func TestHashesEquivalent(t *testing.T) {
	assert.True(t, hashesEquivalent(nil))
	assert.True(t, hashesEquivalent([]*storage.ComplianceOperatorProfileV2{}))
	assert.True(t, hashesEquivalent([]*storage.ComplianceOperatorProfileV2{
		makeProfile("x", "c1", tailored, "abc"),
	}))
	assert.True(t, hashesEquivalent([]*storage.ComplianceOperatorProfileV2{
		makeProfile("x", "c1", tailored, "abc"),
		makeProfile("x", "c2", tailored, "abc"),
	}))
	assert.True(t, hashesEquivalent([]*storage.ComplianceOperatorProfileV2{
		makeProfile("x", "c1", tailored, ""),
		makeProfile("x", "c2", tailored, ""),
	}))
	assert.False(t, hashesEquivalent([]*storage.ComplianceOperatorProfileV2{
		makeProfile("x", "c1", tailored, "abc"),
		makeProfile("x", "c2", tailored, "xyz"),
	}))
}
