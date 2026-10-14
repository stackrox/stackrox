package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

type options struct {
	rules, defaults, files, labels, comments string
	repo, pr, commit, commitTime, job        string
	decisionOut, logOut                      string
	unknown, enforce                         bool
}

func runLocal(command, mode string, o options, output, diagnostic io.Writer) error {
	if mode != "" && mode != "gate" && !(command == "gha" && mode == "print") {
		return fmt.Errorf("unknown %s mode %q", command, mode)
	}
	if mode == "gate" && o.job == "" {
		return fmt.Errorf("gate needs --job")
	}
	if o.unknown == (o.files != "") {
		return fmt.Errorf("pass either --files-file or --files-unknown")
	}
	files := []string{}
	if !o.unknown {
		text, err := os.ReadFile(o.files)
		if err != nil {
			return errors.Wrap(err, "read changed files")
		}
		files = readList(string(text))
	}
	labels := []string{}
	if o.labels != "" {
		text, err := os.ReadFile(o.labels)
		if err != nil {
			return errors.Wrap(err, "read labels")
		}
		labels = readList(string(text))
	}
	meta := metadata{repo: o.repo, pr: o.pr, commit: o.commit}
	var resolved result
	var d defaults
	var err error
	if command == "resolve" {
		if o.unknown {
			return fmt.Errorf("changed files are unknown")
		}
		m, loadErr := loadMapping(o.rules)
		if loadErr != nil {
			return loadErr
		}
		text, readErr := os.ReadFile(o.defaults)
		if readErr != nil {
			return errors.Wrap(readErr, "read decision-defaults")
		}
		d, err = parseDefaults(string(text))
		if err != nil {
			return err
		}
		resolved, err = resolve(files, labels, m, d, meta)
	} else {
		resolved, d, err = openDecision(o.rules, o.defaults, files, labels, o.unknown, meta)
	}
	if err != nil {
		return err
	}
	if o.decisionOut != "" {
		if err := os.WriteFile(o.decisionOut, []byte(resolved.decision), 0o600); err != nil {
			return errors.Wrap(err, "write decision file")
		}
	}
	if o.logOut != "" {
		if err := os.WriteFile(o.logOut, []byte(resolved.log), 0o600); err != nil {
			return errors.Wrap(err, "write resolver log")
		}
	}
	if command == "resolve" {
		if o.decisionOut == "" {
			_, err = io.WriteString(output, resolved.decision)
		}
		return errors.Wrap(err, "print decision")
	}
	if mode == "print" {
		return printPlan(resolved, o.enforce, output)
	}
	forced := false
	if command == "prow" && o.comments != "" {
		comments, err := readComments(o.comments)
		if err != nil {
			return err
		}
		forced = requestsSince(comments, o.job, o.commitTime)
	}
	if _, err := io.WriteString(diagnostic, resolved.log); err != nil {
		return errors.Wrap(err, "print resolver log")
	}
	action := gateAction(o.job, resolved.jobs, d, o.enforce, forced)
	if !o.enforce {
		fmt.Fprintf(diagnostic, "%s: decision would %s; label ci-dispatcher-enforce is absent so the job runs\n",
			o.job, gateAction(o.job, resolved.jobs, d, true, forced))
	}
	_, err = fmt.Fprintln(output, action)
	return errors.Wrap(err, "print gate action")
}

func validateRules(o options, output io.Writer) error {
	m, err := loadMapping(o.rules)
	if err != nil {
		return err
	}
	text, err := os.ReadFile(o.defaults)
	if err != nil {
		return errors.Wrap(err, "read decision-defaults")
	}
	d, err := parseDefaults(string(text))
	if err != nil {
		return err
	}
	if _, err := resolve(nil, nil, m, d, metadata{}); err != nil {
		return err
	}
	mentioned := map[string]bool{}
	for _, r := range m.Rules {
		for _, names := range [][]string{r.Run, r.Skip, r.Default} {
			for _, job := range names {
				mentioned[job] = true
			}
		}
	}
	for _, job := range m.Jobs {
		if !mentioned[job] {
			fmt.Fprintf(output, "WARNING: %s has no explicit job rule\n", job)
		}
	}
	_, err = fmt.Fprintln(output, "Validation passed")
	return errors.Wrap(err, "print validation result")
}

func rootDir() string {
	if root := os.Getenv("CI_SELECT_ROOT"); root != "" {
		return root
	}
	if root := os.Getenv("ROOT"); root != "" {
		return root
	}
	return "."
}

func rulePaths() (string, string) {
	return filepath.Join(rootDir(), "ci/test-domains.toml"), filepath.Join(rootDir(), "ci/decision-defaults")
}
