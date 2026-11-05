package main

// gha.go reports a selection as GitHub outputs: it reads the pull-request
// event, loads the rules through validation.Load, computes the diff with
// changedFiles (diff.go), resolves it, then writes GITHUB_OUTPUT, the step
// summary, and the enforce label behavior. The diff mechanics live in
// diff.go.

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/resolver"
	"github.com/stackrox/rox/ci/select/internal/validation"
)

const enforceLabel = "ci-dispatcher-enforce"

type pullRequestEvent struct {
	PullRequest *struct {
		Labels []struct{ Name string }
		Base   struct{ SHA string }
		Head   struct{ SHA string }
	} `json:"pull_request"`
}

func ghaDecision(ctx context.Context, root string) (resolver.Decision, bool) {
	if os.Getenv("GITHUB_EVENT_NAME") != "pull_request" {
		return resolver.Fallback(errors.New("selection applies only to pull requests")), false
	}
	data, err := os.ReadFile(os.Getenv("GITHUB_EVENT_PATH"))
	if err != nil {
		return resolver.Fallback(errors.Wrap(err, "read PR event")), false
	}
	var event pullRequestEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return resolver.Fallback(errors.Wrap(err, "decode PR event")), false
	}
	if event.PullRequest == nil {
		return resolver.Fallback(errors.New("event has no pull request")), false
	}
	pr := event.PullRequest
	enforce := slices.ContainsFunc(pr.Labels, func(label struct{ Name string }) bool { return label.Name == enforceLabel })
	// The same linter entry as the CLI: rules are fully checked before
	// anything is resolved.
	m, warnings, err := validation.Load(filepath.Join(root, "ci/test-selection-rules.yaml"))
	if err != nil {
		return resolver.Fallback(errors.New(lintText(err, warnings))), enforce
	}
	files, err := changedFiles(ctx, root, pr.Base.SHA, pr.Head.SHA)
	if err != nil {
		return resolver.Fallback(err), enforce
	}
	return resolver.Resolve(files, m), enforce
}

func gha(ctx context.Context, output, diagnostic io.Writer) error {
	root := os.Getenv("CI_SELECT_ROOT")
	if root == "" {
		root = "."
	}
	d, enforce := ghaDecision(ctx, root)
	var details []string
	applied := make(map[string]string, len(d.Actions))
	for _, target := range slices.Sorted(maps.Keys(d.Actions)) {
		applied[target] = "run"
		if enforce {
			applied[target] = d.Action(target)
		}
		details = append(details, fmt.Sprintf("%s: planned=%s; applied=%s\n", target, d.Action(target), applied[target]))
	}
	// Section headers keep the step summary readable: the decisions list and
	// the trace are separated from each other and from the enforce flag.
	body := fmt.Sprintf("enforce=%t\n-- decisions --\n%s\n-- trace --\n%s%s",
		enforce, d.String(), d.Log, strings.Join(details, ""))
	data, err := json.Marshal(applied)
	if err != nil {
		return errors.Wrap(err, "encode decisions")
	}
	if err := appendFile(os.Getenv("GITHUB_OUTPUT"), "decisions="+string(data)+"\n"); err != nil {
		return err
	}
	fmt.Fprint(output, body)
	return appendFile(os.Getenv("GITHUB_STEP_SUMMARY"), "## CI selection\n\n<pre>"+html.EscapeString(body)+"</pre>\n")
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
