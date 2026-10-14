package main

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type metadata struct {
	repo   string
	pr     string
	commit string
}

type vote struct {
	run          []string
	skip         []string
	defaultRules []string
}

type result struct {
	decision string
	log      string
	jobs     map[string]bool
}

func resolve(files, labels []string, m mapping, d defaults, meta metadata) (result, error) {
	for _, job := range m.Jobs {
		if d.actions[job] == "" {
			return result{}, fmt.Errorf("decision-defaults is missing %s", job)
		}
	}
	for _, job := range d.jobs {
		if !slices.Contains(m.Jobs, job) {
			return result{}, fmt.Errorf("decision-defaults has unknown target %s", job)
		}
	}
	files = dedupe(files)
	votes := map[string]*vote{}
	for _, job := range m.Jobs {
		votes[job] = &vote{}
	}
	for _, r := range m.Rules {
		if r.When != "remaining" && !r.matches(files, labels) {
			continue
		}
		for _, job := range m.Jobs {
			v := votes[job]
			if r.When == "remaining" && len(v.run)+len(v.skip)+len(v.defaultRules) != 0 {
				continue
			}
			if namesJob(r.Run, job) {
				v.run = append(v.run, r.Name)
			}
			if namesJob(r.Skip, job) {
				v.skip = append(v.skip, r.Name)
			}
			if namesJob(r.Default, job) {
				v.defaultRules = append(v.defaultRules, r.Name)
			}
		}
	}
	clashes := map[string]bool{}
	fromDefault := map[string]bool{}
	original := map[string]bool{}
	for _, job := range m.Jobs {
		v := votes[job]
		switch {
		case len(v.run) > 0:
			original[job] = true
			clashes[job] = len(v.skip) > 0
		case len(v.skip) > 0:
		case len(v.defaultRules) > 0:
			original[job] = d.actions[job] == "run"
			fromDefault[job] = true
		default:
			return result{}, fmt.Errorf("%s has no decision", job)
		}
	}
	running := map[string]bool{}
	chosen := []string{}
	for _, job := range m.Jobs {
		if original[job] {
			running[job] = true
			chosen = append(chosen, job)
		}
	}
	type addition struct{ job, because string }
	added := []addition{}
	for changed := true; changed; {
		changed = false
		for _, job := range m.Jobs {
			if !running[job] {
				continue
			}
			for _, needed := range m.Requires[job] {
				if !running[needed] {
					running[needed] = true
					chosen = append(chosen, needed)
					added = append(added, addition{job: needed, because: job})
					changed = true
				}
			}
		}
	}
	events := []string{}
	for _, r := range m.Rules {
		if r.When == "remaining" {
			continue
		}
		matched := false
		actions := []string{}
		for _, job := range m.Jobs {
			v := votes[job]
			run := slices.Contains(v.run, r.Name)
			skip := slices.Contains(v.skip, r.Name)
			matched = matched || run || skip || slices.Contains(v.defaultRules, r.Name)
			if clashes[job] {
				continue
			}
			if run {
				actions = append(actions, fmt.Sprintf("rule %q runs target %q", r.Name, job))
			}
			if skip {
				actions = append(actions, fmt.Sprintf("rule %q skips target %q", r.Name, job))
			}
		}
		if matched {
			if event := r.matchEvent(files); event != "" {
				events = append(events, event)
			}
		}
		events = append(events, actions...)
	}
	for _, job := range m.Jobs {
		if clashes[job] {
			v := votes[job]
			events = append(events, fmt.Sprintf("clash on target %q; %s, %s; run wins",
				job, says(v.run, "run"), says(v.skip, "skip")))
		}
	}
	for _, job := range m.Jobs {
		if fromDefault[job] && (original[job] || !running[job]) {
			events = append(events, fmt.Sprintf("rule %q takes default %q for target %q",
				votes[job].defaultRules[0], d.actions[job], job))
		}
	}
	for _, entry := range added {
		events = append(events, fmt.Sprintf("added target %q because target %q requires it", entry.job, entry.because))
	}
	return result{decision: formatDecision(chosen), log: formatLog(meta, events), jobs: running}, nil
}

func openDecision(rulesPath, defaultsPath string, files, labels []string, unknown bool, meta metadata) (result, defaults, error) {
	text, err := os.ReadFile(defaultsPath)
	if err != nil {
		return result{}, defaults{}, errors.Wrap(err, "read decision-defaults")
	}
	d, err := parseDefaults(string(text))
	if err != nil {
		return result{}, d, err
	}
	m, err := loadMapping(rulesPath)
	if err == nil && unknown {
		err = fmt.Errorf("changed files are unknown")
	}
	if err == nil {
		var resolved result
		resolved, err = resolve(files, labels, m, d, meta)
		if err == nil {
			return resolved, d, nil
		}
	}
	decision := d.decision()
	jobs, parseErr := parseDecision(decision)
	if parseErr != nil {
		return result{}, d, parseErr
	}
	return result{
		decision: decision,
		jobs:     jobs,
		log:      fmt.Sprintf("resolver failed: %v\nusing decision-defaults\n", err),
	}, d, nil
}

func namesJob(names []string, job string) bool {
	return slices.Contains(names, "*") || slices.Contains(names, job)
}

func (r rule) matches(files, labels []string) bool {
	switch r.When {
	case "always":
		return true
	case "no-file-changed":
		return len(files) == 0
	case "label-exists":
		return slices.Contains(labels, r.Label)
	case "any-file-matches":
		return slices.ContainsFunc(files, r.matchesPath)
	case "every-file-matches":
		return len(files) > 0 && !slices.ContainsFunc(files, func(path string) bool { return !r.matchesPath(path) })
	default:
		return false
	}
}

func (r rule) matchesPath(path string) bool {
	for _, pattern := range r.patterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

func (r rule) matchEvent(files []string) string {
	switch r.When {
	case "label-exists":
		return fmt.Sprintf("rule %q matches because label %q is set", r.Name, r.Label)
	case "no-file-changed":
		return fmt.Sprintf("rule %q matches because the diff has no files", r.Name)
	case "any-file-matches", "every-file-matches":
	default:
		return ""
	}
	groups := make([][]string, len(r.patterns))
	matched := 0
	for _, path := range files {
		for index, pattern := range r.patterns {
			if pattern.MatchString(path) {
				groups[index] = append(groups[index], path)
				matched++
				break
			}
		}
	}
	if matched == 0 {
		return ""
	}
	header := fmt.Sprintf("rule %q matches", r.Name)
	if r.When == "every-file-matches" {
		header += " every changed file"
	} else if matched > 1 && matched == len(files) {
		header += fmt.Sprintf(" all %d changed files", matched)
	}
	lines := []string{header}
	for index, paths := range groups {
		if len(paths) == 0 {
			continue
		}
		kind, label := patternKind(r.Paths[index])
		switch {
		case kind == "exact":
			lines = append(lines, "     "+label)
		case kind == "prefix" && matched != 1:
			lines = append(lines, fmt.Sprintf("     %s (%d)", label, len(paths)))
		default:
			for _, path := range paths {
				lines = append(lines, "     "+path)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func patternKind(source string) (string, string) {
	body := strings.TrimPrefix(source, "^")
	body = strings.TrimSuffix(body, "$")
	var literal strings.Builder
	for index := 0; index < len(body); index++ {
		char := body[index]
		if char == '\\' {
			index++
			if index == len(body) {
				return "other", source
			}
			literal.WriteByte(body[index])
			continue
		}
		if strings.ContainsRune(".+*?[](){}|", rune(char)) {
			return "other", source
		}
		literal.WriteByte(char)
	}
	if strings.HasPrefix(source, "^") {
		if strings.HasSuffix(source, "$") {
			return "exact", literal.String()
		}
		return "prefix", literal.String()
	}
	return "other", literal.String()
}

func says(names []string, action string) string {
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, "rule "+strconv.Quote(name))
	}
	verb := "says"
	if len(names) > 1 {
		verb = "say"
	}
	return strings.Join(parts, " and ") + " " + verb + " " + action
}

func formatLog(meta metadata, events []string) string {
	var log strings.Builder
	commit := meta.commit
	if len(commit) > 7 {
		commit = commit[:7]
	}
	fmt.Fprintf(&log, "# %s PR %s %s\n# %s\n", meta.repo, meta.pr, commit, time.Now().Format(time.RFC3339))
	width := max(3, len(strconv.Itoa(len(events))))
	for index, event := range events {
		fmt.Fprintf(&log, "%0*d. %s\n", width, index+1, event)
	}
	return log.String()
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			unique = append(unique, value)
		}
	}
	return unique
}
