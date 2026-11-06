// Package engine is the seam between selection callers and a rules
// language. One Engine implementation owns one language end to end:
// the document syntax, the well-formedness it needs to evaluate
// faithfully, and the decision semantics. Callers — the CLI and the
// GitHub Actions planner — depend only on this package, so replacing
// the language means adding an implementation, not changing callers.
package engine

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Decision is the outcome of one selection run: one action ("run" or
// "skip") per CI target, plus the human-readable trace explaining
// every decision. It is engine-agnostic by contract: implementations
// must express their verdict through it, extending it additively
// rather than reinterpreting fields.
type Decision struct {
	// Actions maps a target to "run" or "skip". Targets left undecided
	// are absent; Action reports "run" for them, so consumers without
	// a target inventory fail safe.
	Actions map[string]string
	// Log explains the decisions for humans.
	Log string
}

// Action reports the action for one target; anything undecided runs.
func (d Decision) Action(target string) string {
	if action := d.Actions[target]; action != "" {
		return action
	}
	return "run"
}

func (d Decision) String() string {
	if d.Actions == nil {
		return "all targets run\n"
	}
	var text strings.Builder
	for _, job := range slices.Sorted(maps.Keys(d.Actions)) {
		fmt.Fprintf(&text, "%s %s\n", job, d.Actions[job])
	}
	return text.String()
}

// Fallback is the fail-closed decision: selection could not run, so
// every target runs, and the trace says why.
func Fallback(err error) Decision {
	return Decision{Log: fmt.Sprintf("selection unavailable: %v; all targets run\n", err)}
}

// Rules is a rules document compiled for one Engine. It is an alias of
// any on purpose: only the Engine that produced it can Resolve it, and
// it cannot be sealed at the type level, because Go restricts
// unexported interface methods to the defining package. Callers must
// treat the value as opaque; implementations type-assert their private
// representation and fall back on anything else.
type Rules = any

// Engine implements one rules language behind the selection callers.
type Engine interface {
	// Compile parses one rules document and prepares it for Resolve.
	// It must reject anything it cannot evaluate faithfully — malformed
	// input, unknown versions, uncompilable patterns — so fail-closed
	// never depends on which validation implementation ran.
	Compile(data []byte) (Rules, error)

	// Resolve decides run or skip for every target given the changed
	// paths. rules must come from Compile of this implementation.
	Resolve(files []string, rules Rules) Decision
}
