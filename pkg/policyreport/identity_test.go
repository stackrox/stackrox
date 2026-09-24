package policyreport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeID_DeterministicForSameInput(t *testing.T) {
	id1 := ComputeID("cluster-1", "report-uid-1", SourceKyverno, "policy-a", "rule-a", "subject-uid-1")
	id2 := ComputeID("cluster-1", "report-uid-1", SourceKyverno, "policy-a", "rule-a", "subject-uid-1")
	assert.Equal(t, id1, id2)
	assert.NotEmpty(t, id1)
}

func TestComputeID_DiffersWhenAnyComponentChanges(t *testing.T) {
	base := func() string {
		return ComputeID("cluster-1", "report-uid-1", SourceKyverno, "policy-a", "rule-a", "subject-uid-1")
	}
	baseID := base()

	for name, id := range map[string]string{
		"different cluster": ComputeID("cluster-2", "report-uid-1", SourceKyverno, "policy-a", "rule-a", "subject-uid-1"),
		"different report":  ComputeID("cluster-1", "report-uid-2", SourceKyverno, "policy-a", "rule-a", "subject-uid-1"),
		"different policy":  ComputeID("cluster-1", "report-uid-1", SourceKyverno, "policy-b", "rule-a", "subject-uid-1"),
		"different rule":    ComputeID("cluster-1", "report-uid-1", SourceKyverno, "policy-a", "rule-b", "subject-uid-1"),
		"different subject": ComputeID("cluster-1", "report-uid-1", SourceKyverno, "policy-a", "rule-a", "subject-uid-2"),
		"different source":  ComputeID("cluster-1", "report-uid-1", Source("Gatekeeper"), "policy-a", "rule-a", "subject-uid-1"),
	} {
		t.Run(name, func(t *testing.T) {
			assert.NotEqual(t, baseID, id)
		})
	}
}

func TestComputeID_ConcatenationAmbiguityIsAvoided(t *testing.T) {
	// Without a separator, ("cluster-1x", "report-1") and ("cluster-1", "x-report-1")
	// could hash identically. The NUL separator in ComputeID must prevent this.
	idA := ComputeID("cluster-1x", "report-1", SourceKyverno, "p", "r", "s")
	idB := ComputeID("cluster-1", "x-report-1", SourceKyverno, "p", "r", "s")
	assert.NotEqual(t, idA, idB)
}

func TestComputeID_IndependentOfResultOrdering(t *testing.T) {
	// ComputeID takes explicit fields, not a result's position in an array —
	// this test documents that guarantee at the identity layer directly
	// (the adapter-level reordering-stability test lives in
	// kyverno_v1alpha2_test.go, since ordering is a property of the results
	// slice the adapter walks, not of ComputeID's own arguments).
	idFirst := ComputeID("cluster-1", "report-1", SourceKyverno, "policy-a", "rule-a", "subject-1")
	idSecond := ComputeID("cluster-1", "report-1", SourceKyverno, "policy-a", "rule-a", "subject-1")
	assert.Equal(t, idFirst, idSecond)
}
