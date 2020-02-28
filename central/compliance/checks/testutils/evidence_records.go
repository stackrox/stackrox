package testutils

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stretchr/testify/assert"
)

// EvidenceRecords contains a slice of evidencerecords, as well as some utility methods.
type EvidenceRecords struct {
	List []framework.EvidenceRecord
}

// AssertExpectedResult asserts that the evidence records have the expected result.
func (e *EvidenceRecords) AssertExpectedResult(expectedToPass bool, t *testing.T) {
	if expectedToPass {
		e.CheckPassed(t)
	} else {
		e.CheckFailed(t)
	}
}

// CheckPassed verifies that the evidence records have passed.
func (e *EvidenceRecords) CheckPassed(t *testing.T) {
	assert.NotEmpty(t, e.List)
	for _, r := range e.List {
		assert.Equal(t, framework.PassStatus, r.Status, "Found non-passing record: %+v", r)
	}
}

// CheckFailed verifies that the evidence records have failed.
// It also verifies that there is no state other than pass or fail.
func (e *EvidenceRecords) CheckFailed(t *testing.T) {
	var atLeastOneFail bool
	for _, r := range e.List {
		if r.Status == framework.FailStatus {
			atLeastOneFail = true
			continue
		}
		assert.Equal(t, framework.PassStatus, r.Status, "Found record which was neither pass nor fail: %+v", r)
	}
	assert.True(t, atLeastOneFail, "No failing statuses found (got %+v)", e.List)
}
