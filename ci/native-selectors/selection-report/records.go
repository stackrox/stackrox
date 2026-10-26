package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"html"
	"slices"
	"strconv"
	"strings"
	"time"
)

const recordPrefix = "CI selection: "

type (
	pullRequest struct {
		Head struct {
			SHA  string `json:"sha"`
			Ref  string `json:"ref"`
			Repo struct {
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"head"`
	}
	workflowRun struct {
		ID         int64     `json:"id"`
		WorkflowID int64     `json:"workflow_id"`
		Name       string    `json:"name"`
		HeadSHA    string    `json:"head_sha"`
		HeadBranch string    `json:"head_branch"`
		CreatedAt  time.Time `json:"created_at"`
		HeadRepo   struct {
			FullName string `json:"full_name"`
		} `json:"head_repository"`
		PRs []struct{ Number int } `json:"pull_requests"`
	}
	workflowJob struct {
		ID      int64          `json:"id"`
		Name    string         `json:"name"`
		Status  string         `json:"status"`
		Attempt int            `json:"run_attempt"`
		Steps   []workflowStep `json:"steps"`
	}
	workflowStep struct {
		Name        string    `json:"name"`
		Status      string    `json:"status"`
		Conclusion  string    `json:"conclusion"`
		CompletedAt time.Time `json:"completed_at"`
	}
	// record contains only observed metadata; it never evaluates selection policy.
	record struct {
		id, run                              int64
		workflow, name, source, kind, reason string
		candidate, eligible, enforced        string
		at                                   time.Time
	}
)

func collect(ctx context.Context, a *api, opt options) ([]record, error) {
	prefix := "repos/" + opt.repo
	var pr pullRequest
	if err := a.get(ctx, fmt.Sprintf("%s/pulls/%d", prefix, opt.pr), &pr); err != nil {
		return nil, err
	}
	if pr.Head.SHA != opt.sha {
		return nil, errNewHead
	}
	if pr.Head.Repo.FullName != opt.repo {
		return nil, errors.New("selection comments require a same-repository PR")
	}
	latest := map[int64]workflowRun{}
	for page := 1; ; page++ {
		var runs struct {
			Runs []workflowRun `json:"workflow_runs"`
		}
		if err := a.get(ctx, runPath(opt.repo, opt.sha, page), &runs); err != nil {
			return nil, err
		}
		for _, run := range runs.Runs {
			belongs := slices.ContainsFunc(run.PRs, func(item struct{ Number int }) bool { return item.Number == opt.pr })
			if len(run.PRs) == 0 {
				belongs = run.HeadBranch == pr.Head.Ref && run.HeadRepo.FullName == pr.Head.Repo.FullName
			}
			if !belongs || run.HeadSHA != opt.sha || run.ID <= 0 || run.WorkflowID <= 0 {
				continue
			}
			saved := latest[run.WorkflowID]
			if saved.ID == 0 || run.CreatedAt.After(saved.CreatedAt) || run.CreatedAt.Equal(saved.CreatedAt) && run.ID > saved.ID {
				latest[run.WorkflowID] = run
			}
		}
		if len(runs.Runs) < 100 {
			break
		}
	}
	var result []record
	for _, id := range sortedKeys(latest) {
		run := latest[id]
		// Partial reruns retain decisions from jobs that were not rerun.
		jobs := map[string]workflowJob{}
		for page := 1; ; page++ {
			var data struct {
				Jobs []workflowJob `json:"jobs"`
			}
			if err := a.get(ctx, fmt.Sprintf("%s/actions/runs/%d/jobs?filter=all&per_page=100&page=%d", prefix, run.ID, page), &data); err != nil {
				return nil, err
			}
			for _, job := range data.Jobs {
				saved := jobs[job.Name]
				if saved.ID == 0 || job.Attempt > saved.Attempt || job.Attempt == saved.Attempt && job.ID > saved.ID {
					jobs[job.Name] = job
				}
			}
			if len(data.Jobs) < 100 {
				break
			}
		}
		for _, job := range jobs {
			if row, ok := readRecord(run, job); ok {
				result = append(result, row)
			}
		}
	}
	slices.SortFunc(result, func(a, b record) int {
		return cmp.Or(strings.Compare(a.workflow, b.workflow), strings.Compare(a.name, b.name), cmp.Compare(a.id, b.id))
	})
	return result, nil
}

func sortedKeys(values map[int64]workflowRun) []int64 {
	keys := make([]int64, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// readRecord accepts an executed metadata step even while its test job is still running.
func readRecord(run workflowRun, job workflowJob) (record, bool) {
	row := record{id: job.ID, run: run.ID, workflow: run.Name, source: job.Name, name: job.Name, kind: "pending"}
	hasExplanation := slices.ContainsFunc(job.Steps, func(step workflowStep) bool { return step.Name == "Explain selection" })
	for _, step := range job.Steps {
		data, ok := strings.CutPrefix(step.Name, recordPrefix)
		if !ok {
			continue
		}
		if step.Status != "completed" {
			return row, true
		}
		parts := strings.Split(data, " | ")
		if step.Conclusion != "success" || len(parts) != 5 || !slices.Contains([]string{"permitted", "skipped-by-selection", "blocked-by-execution"}, parts[0]) {
			row.kind, row.reason = "unavailable", "metadata step failed or lacks a valid decision"
			return row, true
		}
		flags := make([]string, 0, 3)
		for i, prefix := range []string{"candidate=", "eligible=", "enforced="} {
			value, ok := strings.CutPrefix(parts[i+2], prefix)
			if !ok || value != "true" && value != "false" {
				row.kind, row.reason = "unavailable", "metadata flags are missing or invalid"
				return row, true
			}
			flags = append(flags, value)
		}
		if strings.TrimSpace(parts[1]) == "" {
			row.kind, row.reason = "unavailable", "metadata target is missing"
			return row, true
		}
		row.kind, row.name, row.at = parts[0], parts[1], step.CompletedAt
		row.candidate, row.eligible, row.enforced = flags[0], flags[1], flags[2]
		return row, true
	}
	if hasExplanation && job.Status == "completed" {
		row.kind, row.reason = "unavailable", "job ended without an executed decision record"
	}
	return row, hasExplanation
}

func render(opt options, rows []record, active bool, deadline time.Time) string {
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n### Native CI selection for [`%.12s`](https://github.com/%s/commit/%s)\n\n", commentMarker, opt.sha, opt.repo, opt.sha)
	if active {
		fmt.Fprintf(&body, "Watching for decisions every %s until **%s UTC**. Tests run independently.\n\n", opt.interval, deadline.UTC().Format("15:04:05"))
	} else if opt.watch > 0 {
		fmt.Fprint(&body, "**Observation window ended.** Later or unreported decisions are not classified; rerun the observer to refresh them.\n\n")
	} else {
		fmt.Fprint(&body, "Read-only snapshot of currently available decision records.\n\n")
	}
	if opt.run > 0 {
		link := fmt.Sprintf("https://github.com/%s/actions/runs/%d", opt.repo, opt.run)
		if opt.job > 0 {
			link += "/job/" + strconv.FormatInt(opt.job, 10)
		}
		fmt.Fprintf(&body, "[Live decision log](%s)\n\n", link)
	}
	if len(rows) == 0 {
		if active {
			fmt.Fprint(&body, "**Waiting for native decision records.** This does not mean that no jobs were selected.\n\n")
		} else {
			fmt.Fprint(&body, "**No native decision records observed.** This does not mean that no jobs were selected.\n\n")
		}
	} else {
		fmt.Fprintln(&body, "| Workflow | Job or workflow call | Decision | Rule matched (candidate) | Execution restrictions permit work (eligible) | Enforcement enabled | Recorded (UTC) |\n| --- | --- | --- | --- | --- | --- | --- |")
		for i, row := range rows {
			at := "—"
			if !row.at.IsZero() {
				at = row.at.UTC().Format("15:04:05")
			}
			label := map[string]string{
				"permitted": "Work permitted; prerequisites still apply", "skipped-by-selection": "Skipped by selection",
				"blocked-by-execution": "Blocked by execution restrictions", "pending": "Decision pending", "unavailable": "Decision unavailable",
			}[row.kind]
			if row.reason != "" {
				label += ": " + row.reason
			}
			name := row.name
			if row.source != "" && row.source != row.name {
				name += " (" + row.source + ")"
			}
			line := fmt.Sprintf("| %s | [%s](https://github.com/%s/actions/runs/%d/job/%d) | %s | %s | %s | %s | %s |\n", markdown(row.workflow), markdown(name), opt.repo, row.run, row.id, markdown(label), yesNo(row.candidate), yesNo(row.eligible), yesNo(row.enforced), at)
			if body.Len()+len(line) > 58000 {
				fmt.Fprintf(&body, "\n%d additional records omitted to keep this comment small; inspect the linked decision log.\n", len(rows)-i)
				break
			}
			body.WriteString(line)
		}
	}
	fmt.Fprintln(&body, "\nRecords come from the native explanation outputs. This reporter never evaluates paths, labels, or job rules. Permission is not proof that tests ran. Jobs skipped before an explanation cannot be assigned a selection reason from their status alone.")
	return body.String()
}

func markdown(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	return strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]", "|", "\\|", "*", "\\*", "_", "\\_", "`", "\\`").Replace(html.EscapeString(value))
}

func yesNo(value string) string {
	if value == "true" {
		return "Yes"
	}
	if value == "false" {
		return "No"
	}
	return "—"
}
