package framework

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
)

// ComplianceContext is the primary interface through which check implementation access compliance data. It also defines
// the scope for the execution of a single check function: every check function runs against a specific compliance
// context instance, and every context instance refers to a specific target object (the cluster as a whole, a node,
// or a deployment).
//
// Contexts are scoped: when a check is run, its main check function is run against the top-level, cluster-scope
// context. Within this check, additional check functions may be run against deployments or nodes. Each such check
// function invocation happens in a newly created (via `ForObject`) context instance, which is a descendant of the
// top-level context.
//
// Through a context, evidence (pass or fail) can be recorded; additionally, in the case of abnormal termination,
// errors may be recorded. This is accomplished by calling `Abort` within a check function.
//
//go:generate mockgen-wrapper
type ComplianceContext interface {
	// StandardName returns the current standard name
	StandardName() string

	// Domain returns the compliance domain.
	Domain() ComplianceDomain
	// Data returns an interface to the data repository.
	Data() ComplianceDataRepository

	// ForObject creates a child context for the given compliance target.
	ForObject(target ComplianceTarget) ComplianceContext

	// Target is the active compliance target.
	Target() ComplianceTarget

	// RecordEvidence adds an evidence record with the given status and message.
	RecordEvidence(status Status, message string)

	// Finalize is called by the framework after a run for a single target object.
	Finalize(err error)
}

func newToplevelContext(
	standardName string,
	domain ComplianceDomain,
	data ComplianceDataRepository,
	rootResults *results,
	stopSig *concurrency.ErrorSignal) *baseContext {
	ctx := &baseContext{
		standardName: standardName,
		data:         data,
		domain:       domain,
		stopSig:      stopSig,
		target:       domain.Cluster(),
		results:      rootResults,
	}
	return ctx
}

type baseContext struct {
	standardName string
	data         ComplianceDataRepository
	domain       ComplianceDomain
	stopSig      *concurrency.ErrorSignal

	target  ComplianceTarget
	results *results
}

func (c *baseContext) StandardName() string {
	c.checkErr()
	return c.standardName
}

func (c *baseContext) Domain() ComplianceDomain {
	c.checkErr()
	return c.domain
}

func (c *baseContext) Data() ComplianceDataRepository {
	c.checkErr()
	return c.data
}

func (c *baseContext) Target() ComplianceTarget {
	c.checkErr()
	return c.target
}

func (c *baseContext) RecordEvidence(status Status, evidence string) {
	c.checkErr()
	c.results.recordEvidence(status, evidence)
}

func (c *baseContext) Abort(err error) {
	c.checkErr()
	halt(err)
}

func (c *baseContext) ForObject(target ComplianceTarget) ComplianceContext {
	c.checkErr()
	return &baseContext{
		data:    c.data,
		domain:  c.domain,
		stopSig: c.stopSig,
		target:  target,
		results: c.results.forChild(target),
	}
}

func (c *baseContext) checkErr() {
	if err, ok := c.stopSig.Error(); ok {
		halt(errors.Wrap(err, "compliance run was aborted"))
	}
}

func (c *baseContext) Finalize(err error) {
	if err != nil {
		c.results.recordError(err)
	}
}
