package validation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stackrox/rox/ci/select/internal/rules"
	"github.com/stackrox/rox/pkg/set"
)

// checkStructure returns every structural problem in a stable order; empty
// means sound.
func checkStructure(m rules.Mapping) []string {
	var findings []string

	if len(m.Jobs) == 0 {
		findings = append(findings, "jobs inventory must not be empty")
	}

	findings = append(findings, checkRuleShape(m)...)
	findings = append(findings, checkJobCoverage(m)...)
	return findings
}

func checkRuleShape(m rules.Mapping) []string {
	var findings []string
	seen := set.NewStringSet()
	for i, r := range m.Rules {
		label := r.Name
		if label == "" {
			label = fmt.Sprintf("rule #%d", i+1)
			findings = append(findings, fmt.Sprintf("%s has an empty name", label))
		} else if !seen.Add(r.Name) {
			findings = append(findings, fmt.Sprintf("duplicate rule name %q", r.Name))
		}
		if len(r.Jobs) == 0 {
			findings = append(findings, fmt.Sprintf("rule %q lists no jobs", label))
		}
		switch clauseCount(r) {
		case 0:
			findings = append(findings, fmt.Sprintf("rule %q declares none of run_if_changed, skip_if_only_changed, always_run", label))
		case 1:
		default:
			findings = append(findings, fmt.Sprintf("rule %q declares more than one of run_if_changed, skip_if_only_changed, always_run", label))
		}
	}
	return findings
}

// clauseCount counts the mutually exclusive behavior clauses a rule declares;
// exactly one is required.
func clauseCount(r *rules.Rule) int {
	n := 0
	for _, set := range []bool{r.HasRun(), r.HasSkip(), r.AlwaysRun} {
		if set {
			n++
		}
	}
	return n
}

func checkJobCoverage(m rules.Mapping) []string {
	var findings []string
	inventory := set.NewStringSet(m.Jobs...)

	// Track every claimant per job so a double-decided job can name both.
	decidedBy := map[string][]string{}
	record := func(job, where string) {
		decidedBy[job] = append(decidedBy[job], where)
		if !inventory.Contains(job) {
			findings = append(findings, fmt.Sprintf("%s names job %q, which is not in the jobs inventory", where, job))
		}
	}

	for _, r := range m.Rules {
		for _, job := range r.Jobs {
			record(job, fmt.Sprintf("rule %q", r.Name))
		}
	}

	for _, job := range sortedKeys(decidedBy) {
		if places := decidedBy[job]; len(places) > 1 {
			findings = append(findings, fmt.Sprintf("job %q is decided more than once: %s", job, strings.Join(places, ", ")))
		}
	}
	for _, job := range m.Jobs {
		if _, ok := decidedBy[job]; !ok {
			findings = append(findings, fmt.Sprintf("job %q is in the inventory but no rule decides it", job))
		}
	}
	return findings
}

func sortedKeys(byJob map[string][]string) []string {
	keys := make([]string, 0, len(byJob))
	for job := range byJob {
		keys = append(keys, job)
	}
	sort.Strings(keys)
	return keys
}
