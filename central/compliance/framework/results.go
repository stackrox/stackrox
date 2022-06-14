package framework

import (
	"github.com/stackrox/rox/pkg/sync"
)

// EvidenceRecord stores the evidence from a compliance check for a single target.
type EvidenceRecord struct {
	Status  Status
	Message string
}

// Results provides access to the results of a compliance check. Like `Context`s, Results objects are scoped, i.e.,
// the top-level Results object stores the cluster objects, and the results of the child objects (like nodes and
// deployments) can be accessed via `ForChild`.
// Note that while evidence should only be recorded at the scope at which the check is run (i.e., only for deployment
// targets in case of a deployment-scope check), errors can be recorded at any scope that is touched by the check.
type Results interface {
	// Error returns the error that occurred while executing the check, if any.
	Error() error
	// Evidence returns a list of evidence records (along with pass/fail statuses) for the object in scope.
	Evidence() []EvidenceRecord

	// ForChild obtains the Results for a child object (deployments/nodes at cluster scope). If no check was executed
	// for the given child, `nil` is returned.
	ForChild(target ComplianceTarget) Results
}

type results struct {
	mutex           sync.Mutex
	err             error
	evidenceRecords []EvidenceRecord
	childResults    map[TargetRef]*results
}

func newResults() *results {
	return &results{
		childResults: make(map[TargetRef]*results),
	}
}

func (r *results) Error() error {
	return r.err
}

func (r *results) Evidence() []EvidenceRecord {
	return r.evidenceRecords
}

func (r *results) ForChild(target ComplianceTarget) Results {
	childRes := r.childResults[GetTargetRef(target)]
	if childRes == nil {
		return nil
	}
	return childRes
}

func (r *results) recordEvidence(status Status, message string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.evidenceRecords = append(r.evidenceRecords, EvidenceRecord{
		Status:  status,
		Message: message,
	})
}

func (r *results) recordError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.err = err
}

func (r *results) forChild(target ComplianceTarget) *results {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ref := GetTargetRef(target)

	childResults := r.childResults[ref]
	if childResults == nil {
		childResults = newResults()
		r.childResults[ref] = childResults
	}

	return childResults
}
