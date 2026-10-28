package main

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

type decision struct {
	actions map[string]string
	log     string
}

func resolve(files []string, m mapping) decision {
	actions := make(map[string]string, len(m.Jobs))
	var log strings.Builder
	for _, path := range files {
		fmt.Fprintf(&log, "changed path: %q\n", path)
	}
	if len(files) == 0 {
		log.WriteString("empty diff: no file rule matches\n")
	}
	for _, r := range m.Rules {
		if r.When == "remaining" {
			for _, job := range m.Jobs {
				if actions[job] == "" {
					actions[job] = "run"
					fmt.Fprintf(&log, "%s: run by %q\n", job, r.Name)
				}
			}
			continue
		}
		if len(files) == 0 || slices.ContainsFunc(files, func(path string) bool {
			return !slices.ContainsFunc(r.patterns, func(pattern *regexp.Regexp) bool {
				return pattern.MatchString(path)
			})
		}) {
			continue
		}
		for _, job := range m.Jobs {
			if slices.Contains(r.Skip, "*") || slices.Contains(r.Skip, job) {
				actions[job] = "skip"
				fmt.Fprintf(&log, "%s: skip by %q (every changed path matches)\n", job, r.Name)
			}
		}
	}
	return decision{actions: actions, log: log.String()}
}

func fallback(err error) decision {
	return decision{log: fmt.Sprintf("selection unavailable: %v; all targets run\n", err)}
}

func (d decision) action(job string) string {
	if action := d.actions[job]; action != "" {
		return action
	}
	return "run"
}

func (d decision) String() string {
	if d.actions == nil {
		return "all targets run\n"
	}
	var text strings.Builder
	for _, job := range slices.Sorted(maps.Keys(d.actions)) {
		fmt.Fprintf(&text, "%s %s\n", job, d.actions[job])
	}
	return text.String()
}
