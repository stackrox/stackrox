package main

// Trigger is the reason a CI job runs, generalized across Prow and GitHub
// Actions so a single vocabulary can describe jobs from either system.
type Trigger int

const (
	// TriggerUnknown means the trigger could not be determined.
	TriggerUnknown Trigger = iota
	// TriggerPreMerge covers Prow presubmits and GitHub Actions pull_request
	// (and pull_request_target) triggers.
	TriggerPreMerge
	// TriggerPostMerge covers Prow postsubmits (steps.postsubmit: true) and
	// GitHub Actions push triggers.
	TriggerPostMerge
	// TriggerPeriodic covers Prow periodics (jobs with a cron:) and GitHub
	// Actions schedule triggers.
	TriggerPeriodic
	// TriggerNightly covers any Prow test defined only under the "nightlies"
	// ci-operator branch/variant. Mechanically these are a mix of
	// postsubmits (gated on a push to the nightlies branch, which
	// .github/workflows/nightly-tag.yml recreates on a schedule) and true
	// periodics (cron:) -- but from a human "when does this run" standpoint
	// they're both "roughly once a day, independent of any single PR or
	// merge," which is a distinct enough category from a normal postsubmit
	// or periodic to call out on its own.
	TriggerNightly
	// TriggerManual covers GitHub Actions workflow_dispatch triggers. Prow
	// has no clean equivalent, since presubmits with always_run: false are
	// still presubmits, just opt-in via a /test comment.
	TriggerManual
	// TriggerReusable covers GitHub Actions workflow_call triggers: the
	// workflow never runs on its own, only when another workflow calls it.
	TriggerReusable
)

// String returns the short, stable name used in JSON output.
func (t Trigger) String() string {
	switch t {
	case TriggerPreMerge:
		return "pre-merge"
	case TriggerPostMerge:
		return "post-merge"
	case TriggerPeriodic:
		return "periodic"
	case TriggerNightly:
		return "nightly"
	case TriggerManual:
		return "manual"
	case TriggerReusable:
		return "reusable"
	default:
		return "unknown"
	}
}

// Description returns a longer-form explanation of what the trigger means,
// suitable for display next to a job in the matrix.
func (t Trigger) Description() string {
	switch t {
	case TriggerPreMerge:
		return "Runs automatically against proposed changes (a pull request) before they can merge."
	case TriggerPostMerge:
		return "Runs after a change has already merged, to validate the resulting merge commit."
	case TriggerPeriodic:
		return "Runs on a fixed schedule, independent of any code change."
	case TriggerNightly:
		return "Runs roughly once a day against a nightly build, independent of any single pull request or merge."
	case TriggerManual:
		return "Only runs when a person explicitly triggers it."
	case TriggerReusable:
		return "Never runs on its own; only runs when another workflow calls it."
	default:
		return "The trigger for this job could not be determined."
	}
}

// MarshalJSON renders the Trigger as its short string name.
func (t Trigger) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}
