package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"html"
	"slices"
	"strings"
	"time"
)

const (
	decisionMarker = "STACKROX_CI_SELECTION_V1 "
	selectionSkip  = "skipped-by-selection"
	executionBlock = "blocked-by-execution"
	noDecision     = "no-recorded-decision"
)

type (
	// selectionReport stores a workflow's observed exceptions; repeated contributions replace it.
	selectionReport struct {
		RunID    int64         `json:"run"`
		Recorded int           `json:"recorded"`
		Jobs     []jobDecision `json:"jobs,omitempty"`
	}
	// jobDecision describes recorded suppression or a skipped/instrumented job lacking a record.
	jobDecision struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Kind   string `json:"kind"`
		Reason string `json:"reason,omitempty"`
	}
)

// collectDecisions reads emitted results, never path rules, labels, or inferred skip reasons.
func collectDecisions(ctx context.Context, request requestFunc, prefix string, runID int64, jobs []workflowJob) selectionReport {
	report := selectionReport{RunID: runID}
	for _, job := range jobs {
		if slices.Contains(strings.Split(job.Name, " / "), "CI timing report") || job.Conclusion == "cancelled" {
			continue
		}
		entry := jobDecision{ID: job.ID, Name: job.Name, Kind: noDecision}
		step := slices.IndexFunc(job.Steps, func(step workflowStep) bool { return step.Name == "Explain selection" })
		if job.Conclusion == "skipped" {
			entry.Reason = "job skipped before recording a decision"
		} else if step == -1 {
			continue
		} else if job.Steps[step].Conclusion != "success" {
			entry.Reason = "explanation did not complete successfully"
		} else {
			data, err := request(ctx, "GET", fmt.Sprintf("%s/actions/jobs/%d/logs", prefix, job.ID), nil)
			if err != nil {
				entry.Reason = "decision log unavailable"
			} else {
				kind, err := decisionFromLog(data)
				if err != nil {
					entry.Reason = err.Error()
				} else {
					report.Recorded++
					if kind == "permitted" {
						continue
					}
					entry.Kind = kind
				}
			}
		}
		report.Jobs = append(report.Jobs, entry)
	}
	slices.SortFunc(report.Jobs, func(a, b jobDecision) int {
		return cmp.Or(strings.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
	})
	return report
}

// decisionFromLog accepts one timestamped record and ignores echoed shell source.
func decisionFromLog(data []byte) (string, error) {
	result := ""
	for line := range strings.SplitSeq(string(data), "\n") {
		timestamp, message, ok := strings.Cut(strings.TrimSuffix(line, "\r"), " ")
		if !ok {
			continue
		}
		kind, ok := strings.CutPrefix(message, decisionMarker)
		if !ok {
			continue
		}
		if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
			continue
		}
		if result != "" {
			return "", errors.New("multiple decision records")
		}
		switch kind {
		case selectionSkip, executionBlock, "permitted":
			result = kind
		default:
			return "", errors.New("invalid decision record")
		}
	}
	if result == "" {
		return "", errors.New("decision record missing from log excerpt")
	}
	return result, nil
}

// renderDecisions links observed exceptions to their jobs without claiming tests ran or time was saved.
func renderDecisions(body *strings.Builder, repo string, row measurement) {
	fmt.Fprintf(body, "\n<details>\n<summary>%.12s: selection decisions for the initial checks</summary>\n\n", row.SHA)
	if len(row.Reports) == 0 {
		fmt.Fprintln(body, "No decision summary was recorded for this measurement.")
	} else {
		recorded := 0
		for _, report := range row.Reports {
			recorded += report.Recorded
		}
		fmt.Fprintf(body, "Received %d decision records from %d reporting workflows.\n\n", recorded, len(row.Reports))
		for _, category := range []struct{ kind, title string }{
			{selectionSkip, "Skipped by selection"},
			{executionBlock, "Blocked by execution"},
			{noDecision, "No recorded decision"},
		} {
			var entries []string
			for _, report := range row.Reports {
				for _, job := range report.Jobs {
					if job.Kind != category.kind {
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
		fmt.Fprintln(body, "No recorded decision covers skipped jobs and jobs whose explanation failed or could not be read. Other jobs without selection instrumentation are omitted. Permission to run is not proof that tests ran; these lists do not estimate time saved.")
	}
	fmt.Fprintln(body, "\n</details>")
}
