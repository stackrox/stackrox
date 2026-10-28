package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

type defaults struct {
	jobs    []string
	actions map[string]string
}

func readList(text string) []string {
	lines := []string{}
	for line := range strings.SplitSeq(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func parseDefaults(text string) (defaults, error) {
	result := defaults{jobs: []string{}, actions: map[string]string{}}
	for index, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "#") {
			return result, fmt.Errorf("line %d mixes a target and a comment", index+1)
		}
		words := strings.Fields(line)
		if len(words) < 2 {
			return result, fmt.Errorf("line %d needs a target and a decision", index+1)
		}
		action := words[len(words)-1]
		if action != "run" && action != "skip" {
			return result, fmt.Errorf("line %d decision must be run or skip", index+1)
		}
		// Preserve spaces within check names; only the final word is the action.
		job := strings.TrimSpace(line[:len(line)-len(action)])
		if _, exists := result.actions[job]; exists {
			return result, fmt.Errorf("duplicate default for %s", job)
		}
		result.jobs = append(result.jobs, job)
		result.actions[job] = action
	}
	if len(result.jobs) == 0 {
		return result, errors.New("decision-defaults lists no targets")
	}
	return result, nil
}

func parseDecision(text string) (map[string]bool, error) {
	jobs := map[string]bool{}
	if strings.TrimSpace(text) == "" {
		return jobs, errors.New("empty decision")
	}
	for index, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "#") {
			return jobs, fmt.Errorf("line %d mixes a job name and a comment", index+1)
		}
		jobs[line] = true
	}
	return jobs, nil
}

func formatDecision(jobs []string) string {
	if len(jobs) == 0 {
		return "# no job runs\n"
	}
	return strings.Join(jobs, "\n") + "\n"
}

func (d defaults) decision(labels []string) string {
	jobs := []string{}
	forceAll := slices.Contains(labels, "ci-run-all-tests")
	for _, job := range d.jobs {
		if d.actions[job] == "run" || forceAll || slices.Contains(labels, job) {
			jobs = append(jobs, job)
		}
	}
	return formatDecision(jobs)
}

func gateAction(job string, jobs map[string]bool, d defaults, enforce, forced bool) string {
	if !enforce || d.actions[job] == "" || forced {
		return "run"
	}
	if jobs == nil {
		return d.actions[job]
	}
	if jobs[job] {
		return "run"
	}
	return "skip"
}
