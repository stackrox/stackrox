package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

const dataMarker = "<!-- ci-timing-data:"

var shaPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// measurement stores the two timestamps aggregated for a commit in the bot comment.
type measurement struct {
	SHA   string    `json:"sha"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// measure includes failures and green skips without requiring an allocated runner.
func measure(run workflowRun, jobs []workflowJob) (measurement, error) {
	result := measurement{SHA: run.HeadSHA, Start: run.CreatedAt}
	for _, job := range jobs {
		if slices.Contains(strings.Split(job.Name, " / "), "CI timing report") || job.Conclusion == "cancelled" {
			continue
		}
		if job.Status != "completed" || job.CompletedAt.IsZero() {
			return measurement{}, fmt.Errorf("check %q has not finished or lacks a completion timestamp", job.Name)
		}
		if job.CompletedAt.After(result.End) {
			result.End = job.CompletedAt
		}
	}
	if !result.valid() {
		return measurement{}, errors.New("workflow requires a commit, creation timestamp, and completed non-reporting checks")
	}
	return result, nil
}

func (m measurement) valid() bool {
	return shaPattern.MatchString(m.SHA) && !m.Start.IsZero() && !m.End.IsZero() && !m.End.Before(m.Start)
}

// render merges each contribution by earliest start and latest finish, so arrival order does not matter.
func render(repo, old string, current measurement) (string, error) {
	var rows []measurement
	if old != "" {
		_, encoded, ok := strings.Cut(old, dataMarker)
		if !ok {
			return "", errors.New("marked bot comment lacks timestamp data; refusing to overwrite it")
		}
		encoded, _, ok = strings.Cut(encoded, " -->")
		if !ok {
			return "", errors.New("unterminated timestamp data in bot comment")
		}
		if err := json.Unmarshal([]byte(encoded), &rows); err != nil {
			return "", fmt.Errorf("decode comment timestamps: %w", err)
		}
	}
	for _, row := range append(rows, current) {
		if !row.valid() {
			return "", errors.New("comment timestamps require a commit, start, and finish in chronological order")
		}
	}
	index := slices.IndexFunc(rows, func(row measurement) bool { return row.SHA == current.SHA })
	if index == -1 {
		rows = append(rows, current)
	} else {
		if current.Start.Before(rows[index].Start) {
			rows[index].Start = current.Start
		}
		if current.End.After(rows[index].End) {
			rows[index].End = current.End
		}
	}
	slices.SortFunc(rows, func(a, b measurement) int {
		return cmp.Or(a.Start.Compare(b.Start), cmp.Compare(a.SHA, b.SHA))
	})
	var body strings.Builder
	fmt.Fprintln(&body, commentMarker)
	fmt.Fprintln(&body, "### GitHub Actions wall-clock wait\n\n| Commit | Start (UTC) | Last check finished (UTC) | Wall-clock wait |\n| --- | --- | --- | --- |")
	for _, row := range rows {
		fmt.Fprintf(&body, "| [`%.12s`](https://github.com/%s/commit/%s) | %s | %s | %s |\n", row.SHA, repo, row.SHA,
			row.Start.UTC().Format("2006-01-02 15:04:05"), row.End.UTC().Format("2006-01-02 15:04:05"), row.End.Sub(row.Start).Round(time.Second))
	}
	fmt.Fprintln(&body, "\nWall-clock wait = last check completion − first workflow creation. Includes runner queues, dependency waits, failures, and green skips. Timing reporters and canceled work are excluded.")
	fmt.Fprintln(&body, "\nValues are provisional while the initial GHA checks and their reporters are still running. Only first attempts on PR opening or a new commit contribute; later label events and manual reruns leave these rows unchanged.")
	encoded, err := json.Marshal(rows)
	if err != nil {
		return "", fmt.Errorf("encode comment timestamps: %w", err)
	}
	fmt.Fprintf(&body, "\n%s%s -->\n", dataMarker, encoded)
	return body.String(), nil
}
