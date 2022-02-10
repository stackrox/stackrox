package framework

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// ComplianceRun encapsulates a compliance run (i.e., multiple checks on a given domain). A ComplianceRun can only be
// used for a single run, i.e., the `Run` method may only be called once; afterwards, it may only be used for collecting
// results.
type ComplianceRun interface {
	// Run starts the compliance run, executing all checks. This method blocks until all checks have completed. It
	// returns an error if the entire run has been aborted (either via `Terminate`, or via an error in the parent
	// context).
	Run(ctx context.Context, standardName string, domain ComplianceDomain, data ComplianceDataRepository) error

	// Wait blocks until the compliance run has completed, and returns the final status.
	Wait() error

	// Terminate terminates a compliance run. It returns true if the Terminate invocation actually terminated the run,
	// false if the run was already terminated.
	Terminate(err error) bool

	// GetAllResults returns a map mapping the ID of each check that was run to its corresponding Results.
	GetAllResults() map[string]Results

	GetChecks() []Check
}

type checkRecord struct {
	check   Check
	results *results
}

type complianceRun struct {
	checks  []checkRecord
	stopSig concurrency.ErrorSignal
}

func newComplianceRun(checks ...Check) (*complianceRun, error) {
	checkRecords := make([]checkRecord, 0, len(checks))
	for _, check := range checks {
		record := checkRecord{
			check:   check,
			results: newResults(),
		}
		checkRecords = append(checkRecords, record)
	}

	return &complianceRun{
		checks:  checkRecords,
		stopSig: concurrency.NewErrorSignal(),
	}, nil
}

// NewComplianceRun creates a new compliance run that will execute the given checks.
func NewComplianceRun(checks ...Check) (ComplianceRun, error) {
	return newComplianceRun(checks...)
}

func signalOnContextErr(ctx context.Context, sig *concurrency.ErrorSignal) {
	select {
	case <-ctx.Done():
		sig.SignalWithError(errors.Wrap(ctx.Err(), "context error"))
	case <-sig.Done():
	}
}

func (r *complianceRun) Run(ctx context.Context, standardName string, domain ComplianceDomain, data ComplianceDataRepository) error {
	go signalOnContextErr(ctx, &r.stopSig)

	var wg sync.WaitGroup
	for _, check := range r.checks {
		checkCtx := newToplevelContext(standardName, domain, data, check.results, &r.stopSig)
		wg.Add(1)
		go r.runCheck(checkCtx, check.check.Run, &wg)
	}

	wg.Wait()
	r.stopSig.SignalWithError(nil)

	return r.stopSig.Err()
}

func (r *complianceRun) Wait() error {
	return r.stopSig.Wait()
}

func (r *complianceRun) Terminate(err error) bool {
	return r.stopSig.SignalWithError(err)
}

func (r *complianceRun) runCheck(ctx ComplianceContext, check CheckFunc, wg *sync.WaitGroup) {
	defer wg.Done()
	doRun(ctx, check)
}

func (r *complianceRun) GetAllResults() map[string]Results {
	resultsByCheckID := make(map[string]Results, len(r.checks))
	for _, checkRec := range r.checks {
		resultsByCheckID[checkRec.check.ID()] = checkRec.results
	}
	return resultsByCheckID
}

func (r *complianceRun) GetChecks() []Check {
	checks := make([]Check, len(r.checks))
	for i, check := range r.checks {
		checks[i] = check.check
	}
	return checks
}
