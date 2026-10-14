package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const enforceLabel = "ci-dispatcher-enforce"

func gitOutput(ctx context.Context, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", rootDir()}, args...)...)
	text, err := command.Output()
	return strings.TrimSuffix(string(text), "\n"), err
}

func changedFiles(ctx context.Context, base, head string) ([]string, bool) {
	if base == "" || head == "" {
		return []string{}, true
	}
	text, err := gitOutput(ctx, "diff", "--name-only", "-z", base+"..."+head, "--")
	if err != nil {
		return []string{}, true
	}
	if text == "" {
		return []string{}, false
	}
	return strings.Split(strings.TrimSuffix(text, "\x00"), "\x00"), false
}

func ghaInputs(ctx context.Context) (result, defaults, error) {
	files, unknown := changedFiles(ctx, os.Getenv("CI_BASE"), os.Getenv("CI_HEAD"))
	labels := readList(strings.ReplaceAll(os.Getenv("PR_LABELS"), ",", "\n"))
	rules, defaultsPath := rulePaths()
	return openDecision(rules, defaultsPath, files, labels, unknown, envMetadata())
}

func envMetadata() metadata {
	return metadata{
		repo:   envOr("CI_REPO", envOr("GITHUB_REPOSITORY", "stackrox/stackrox")),
		pr:     envOr("CI_PR", "0"),
		commit: envOr("CI_COMMIT", "unknown"),
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func ghaCI(ctx context.Context, args []string, output, diagnostic io.Writer) error {
	if len(args) > 1 {
		return fmt.Errorf("gha-ci takes at most one job name")
	}
	resolved, d, err := ghaInputs(ctx)
	if err != nil {
		return err
	}
	if path := os.Getenv("CI_DECISION_OUT"); path != "" {
		if err := os.WriteFile(path, []byte(resolved.decision), 0o600); err != nil {
			return errors.Wrap(err, "write decision file")
		}
	}
	enforce := os.Getenv("CI_ENFORCE") == "true"
	if len(args) == 0 || args[0] == "" {
		return printPlan(resolved, enforce, output)
	}
	job := args[0]
	planned := gateAction(job, resolved.jobs, d, true, false)
	action := gateAction(job, resolved.jobs, d, enforce, false)
	fmt.Fprint(diagnostic, resolved.log)
	fmt.Fprintf(diagnostic, "%s: decision would %s; enforce=%t\n", job, planned, enforce)
	return writeOutputs(map[string]string{"action": action}, output)
}

func printPlan(resolved result, enforce bool, output io.Writer) error {
	label := "absent"
	if enforce {
		label = "set"
	}
	body := fmt.Sprintf("ci-dispatcher-enforce is %s.\n\nDecision:\n%s\nLog:\n%s", label, resolved.decision, resolved.log)
	if _, err := io.WriteString(output, body); err != nil {
		return errors.Wrap(err, "print decision plan")
	}
	return appendFile(os.Getenv("GITHUB_STEP_SUMMARY"), "## CI decision\n\n```\n"+body+"```\n")
}

func appendFile(path, text string) error {
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return errors.Wrap(err, "open CI output")
	}
	_, writeErr := io.WriteString(file, text)
	closeErr := file.Close()
	if writeErr != nil {
		return errors.Wrap(writeErr, "write CI output")
	}
	return errors.Wrap(closeErr, "close CI output")
}

func writeOutputs(values map[string]string, output io.Writer) error {
	keys := []string{}
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var text strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&text, "%s=%s\n", key, values[key])
	}
	if _, err := io.WriteString(output, text.String()); err != nil {
		return errors.Wrap(err, "print CI outputs")
	}
	return appendFile(os.Getenv("GITHUB_OUTPUT"), text.String())
}

type dispatchJob struct {
	name           string
	requireLabel   bool
	requireKonflux bool
}

var dispatchJobs = []dispatchJob{
	{name: "e2e-byodb-tests", requireLabel: true},
	{name: "e2e-db-backup-restore-tests"},
	{name: "e2e-gke-upgrade-tests"},
	{name: "e2e-nongroovy-tests-gke"},
	{name: "e2e-nongroovy-tests-gke-konflux", requireKonflux: true},
	{name: "e2e-qa-tests-gke", requireLabel: true},
	{name: "e2e-qa-tests-gke-konflux", requireKonflux: true},
	{name: "e2e-qa-tests-ocp-fips-konflux", requireKonflux: true},
}

type dispatchInput struct {
	pullRequest, fork, draft, uiOnly, enforce bool
	labels                                    []string
}

func legacyDispatch(job dispatchJob, input dispatchInput) bool {
	if input.fork {
		return false
	}
	if !input.pullRequest || slices.Contains(input.labels, job.name) {
		return true
	}
	if job.requireKonflux && !slices.Contains(input.labels, "konflux-build") {
		return false
	}
	return !input.uiOnly && !job.requireLabel && !input.draft
}

func dispatchPlan(input dispatchInput, resolved *result, d defaults) map[string]string {
	jobs := map[string]bool{}
	anyRuns := false
	for _, job := range dispatchJobs {
		runs := legacyDispatch(job, input)
		if input.enforce && resolved != nil && d.actions[job.name] != "" {
			runs = !input.fork && resolved.jobs[job.name]
		}
		jobs[job.name] = runs
		anyRuns = anyRuns || runs
	}
	waitGHA := !input.fork && (!input.pullRequest || !input.draft || anyRuns)
	if input.enforce && resolved != nil && d.actions["wait-for-images"] != "" {
		waitGHA = !input.fork && resolved.jobs["wait-for-images"]
	}
	// The Konflux label is required for its PR builds even when selection is enforced.
	waitKonflux := !input.pullRequest || (slices.Contains(input.labels, "konflux-build") && anyRuns)
	encoded, _ := json.Marshal(jobs)
	return map[string]string{
		"jobs":                    string(encoded),
		"wait-for-gha-images":     strconv.FormatBool(waitGHA),
		"wait-for-konflux-images": strconv.FormatBool(waitKonflux),
	}
}

func ghaDispatchCI(ctx context.Context, output, diagnostic io.Writer) error {
	input := dispatchInput{
		pullRequest: os.Getenv("GITHUB_EVENT_NAME") == "pull_request",
		fork:        os.Getenv("CI_IS_FORK") == "true",
		draft:       os.Getenv("CI_IS_DRAFT") == "true",
		uiOnly:      os.Getenv("CI_UI_ONLY") == "true",
	}
	if input.pullRequest {
		meta := envMetadata()
		labels, err := newGitHubClient().labels(ctx, meta.repo, meta.pr)
		if err != nil {
			return fmt.Errorf("read dispatch labels: %w", err)
		}
		input.labels = labels
		input.enforce = slices.Contains(labels, enforceLabel)
	}
	var resolved *result
	var d defaults
	if input.pullRequest && input.enforce {
		files, unknown := changedFiles(ctx, os.Getenv("CI_BASE"), os.Getenv("CI_HEAD"))
		rules, defaultsPath := rulePaths()
		plan, owned, err := openDecision(rules, defaultsPath, files, input.labels, unknown, envMetadata())
		if err != nil {
			fmt.Fprintf(diagnostic, "CI decision unreadable (%v); using existing dispatch rules\n", err)
		} else {
			resolved, d = &plan, owned
			fmt.Fprint(diagnostic, plan.log)
		}
	}
	return writeOutputs(dispatchPlan(input, resolved, d), output)
}
