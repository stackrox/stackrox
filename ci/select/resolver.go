package main

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/stackrox/rox/pkg/set"
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
		log.WriteString("empty diff: no safe-path coverage\n")
	}
	for _, job := range m.Jobs {
		covered := len(files) > 0
		reasons := set.NewStringSet()
		for _, path := range files {
			pathCovered := false
			for _, r := range m.Rules {
				if r.When == "safe-paths" && slices.Contains(r.Skip, job) &&
					slices.ContainsFunc(r.patterns, func(pattern *regexp.Regexp) bool { return pattern.MatchString(path) }) {
					pathCovered = true
					reasons.Add(r.Name)
				}
			}
			covered = covered && pathCovered
		}
		if covered {
			actions[job] = "skip"
			fmt.Fprintf(&log, "%s: skip by %q (safe rules cover every changed path)\n", job, slices.Sorted(maps.Keys(reasons)))
		}
	}
	for _, r := range m.Rules {
		if r.When != "remaining" {
			continue
		}
		for _, job := range m.Jobs {
			if actions[job] == "" {
				actions[job] = "run"
				fmt.Fprintf(&log, "%s: run by %q\n", job, r.Name)
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
