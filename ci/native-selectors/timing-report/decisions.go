package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"html"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	decisionMarker = "STACKROX_CI_SELECTION_V1 "
	targetMarker   = "STACKROX_CI_SELECTION_TARGET_V1 "
	selectionSkip  = "skipped-by-selection"
	executionBlock = "blocked-by-execution"
	noDecision     = "no-recorded-decision"
)

var targetPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

type (
	// selectionReport stores a workflow's observed exceptions; repeated contributions replace it.
	selectionReport struct {
		RunID             int64         `json:"run"`
		Recorded          int           `json:"recorded"`
		UnattributedSkips int           `json:"unattributed_skips,omitempty"`
		Jobs              []jobDecision `json:"jobs,omitempty"`
	}
	// jobDecision describes recorded suppression or a skipped/instrumented job lacking a record.
	jobDecision struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Kind   string `json:"kind"`
		Reason string `json:"reason,omitempty"`
	}
	// logDecision associates an emitted result with its own job or a named workflow call.
	logDecision struct{ kind, target string }
)

// collectDecisions reads emitted results, never path rules, labels, or inferred skip reasons.
func collectDecisions(ctx context.Context, request requestFunc, prefix string, runID int64, jobs []workflowJob) selectionReport {
	report := selectionReport{RunID: runID}
	var suppressed []string
	for _, job := range jobs {
		if slices.Contains(strings.Split(job.Name, " / "), "CI timing report") || job.Conclusion == "cancelled" {
			continue
		}
		entry := jobDecision{ID: job.ID, Name: job.Name, Kind: noDecision}
		step := slices.IndexFunc(job.Steps, func(step workflowStep) bool { return step.Name == "Explain selection" })
		if step == -1 {
			continue
		} else if job.Conclusion == "skipped" {
			entry.Reason = "job skipped; selection explanation did not run"
		} else if job.Steps[step].Conclusion != "success" {
			entry.Reason = "explanation did not complete successfully"
		} else {
			data, err := request(ctx, "GET", fmt.Sprintf("%s/actions/jobs/%d/logs", prefix, job.ID), nil)
			if err != nil {
				entry.Reason = "decision log unavailable"
			} else {
				decision, err := decisionFromLog(data)
				if err != nil {
					entry.Reason = err.Error()
				} else {
					report.Recorded++
					if decision.kind == "permitted" {
						continue
					}
					entry.Kind = decision.kind
					if decision.target != "" {
						entry.Name = decision.target
						entry.Reason = "recorded by preparation job"
						suppressed = append(suppressed, decision.target)
					}
				}
			}
		}
		report.Jobs = append(report.Jobs, entry)
	}
	for _, job := range jobs {
		if job.Conclusion != "skipped" || slices.Contains(strings.Split(job.Name, " / "), "CI timing report") ||
			slices.ContainsFunc(job.Steps, func(step workflowStep) bool { return step.Name == "Explain selection" }) {
			continue
		}
		if !slices.ContainsFunc(suppressed, func(target string) bool { return job.Name == target || strings.HasPrefix(job.Name, target+" / ") }) {
			report.UnattributedSkips++
		}
	}
	slices.SortFunc(report.Jobs, func(a, b jobDecision) int {
		return cmp.Or(strings.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
	})
	return report
}

// decisionFromLog accepts a timestamped decision and optional target, ignoring echoed shell source.
func decisionFromLog(data []byte) (logDecision, error) {
	var result logDecision
	for line := range strings.SplitSeq(string(data), "\n") {
		timestamp, message, ok := strings.Cut(strings.TrimSuffix(line, "\r"), " ")
		if !ok {
			continue
		}
		if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
			continue
		}
		if target, ok := strings.CutPrefix(message, targetMarker); ok {
			if result.target != "" || !targetPattern.MatchString(target) {
				return logDecision{}, errors.New("invalid or multiple decision targets")
			}
			result.target = target
		}
		if kind, ok := strings.CutPrefix(message, decisionMarker); ok {
			if result.kind != "" {
				return logDecision{}, errors.New("multiple decision records")
			}
			switch kind {
			case selectionSkip, executionBlock, "permitted":
				result.kind = kind
			default:
				return logDecision{}, errors.New("invalid decision record")
			}
		}
	}
	if result.kind == "" {
		return logDecision{}, errors.New("decision record missing from log excerpt")
	}
	return result, nil
}

// renderDecisions links observed exceptions to their jobs without claiming tests ran or time was saved.
func renderDecisions(body *strings.Builder, repo string, row measurement) {
	fmt.Fprintf(body, "\n<details>\n<summary>%.12s: selection decisions for the initial checks</summary>\n\n", row.SHA)
	if row.DetailsOmitted {
		fmt.Fprintln(body, "Decision details omitted to keep this comment within its size limit. Timestamps continue updating; decision details for this commit remain omitted.")
	} else if len(row.Reports) == 0 {
		fmt.Fprintln(body, "No decision summary was recorded for this measurement.")
	} else {
		recorded, permitted, unattributed := 0, 0, 0
		for _, report := range row.Reports {
			recorded += report.Recorded
			permitted += report.Recorded
			unattributed += report.UnattributedSkips
			for _, job := range report.Jobs {
				if job.Kind == selectionSkip || job.Kind == executionBlock {
					permitted--
				}
				if unattributedSkip(job) {
					unattributed++
				}
			}
		}
		fmt.Fprintf(body, "Received %d decision records from %d reporting workflows. **%d decisions permitted work.** This does not confirm that tests ran.\n\n", recorded, len(row.Reports), permitted)
		for _, category := range []struct{ kind, title string }{
			{selectionSkip, "Skipped by selection"},
			{executionBlock, "Blocked by execution"},
			{noDecision, "Explanation step present, decision unavailable"},
		} {
			var entries []string
			for _, report := range row.Reports {
				for _, job := range report.Jobs {
					if job.Kind != category.kind || unattributedSkip(job) {
						continue
					}
					name := strings.Join(strings.Fields(job.Name), " ")
					name = strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_", "`", "\\`").Replace(html.EscapeString(name))
					entry := fmt.Sprintf("[%s](https://github.com/%s/actions/runs/%d/job/%d)", name, repo, report.RunID, job.ID)
					if job.Reason != "" {
						entry += " (" + html.EscapeString(job.Reason) + ")"
					}
					entries = append(entries, entry)
				}
			}
			fmt.Fprintf(body, "**%s (%d)**\n\n", category.title, len(entries))
			if len(entries) == 0 {
				fmt.Fprintln(body, "None in the contributions received so far.")
			} else {
				for _, entry := range entries {
					fmt.Fprintln(body, "- "+entry)
				}
			}
			fmt.Fprintln(body)
		}
		fmt.Fprintf(body, "%d additional jobs were skipped without selection records. Their causes are not identified here and they are not counted as selection skips.\n\n", unattributed)
		fmt.Fprintln(body, "No recorded decision covers jobs whose selection explanation did not run, failed, or could not be read. E2E entries link to the preparation that records and controls the test decision. These lists do not estimate time saved.")
	}
	fmt.Fprintln(body, "\n</details>")
}

// unattributedSkip recognizes status-only records without assuming a selection cause.
func unattributedSkip(job jobDecision) bool {
	return job.Kind == noDecision && job.Reason == "job skipped before recording a decision"
}
